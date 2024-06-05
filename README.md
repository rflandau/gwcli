# gwcli

A redesigned Gravwell client for terminal, supporting both TUI-served interactivity and non-interactive script calls. 

# Design

## Terminology

Bubble Tea has the `tea.Model` interface that must be implemented by a model struct of our own. Bubbles.TextInput is a tea.Model under the hood. Cobra is composed of `cobra.Commands` and Bubble Tea drives its I/O via `tea.Cmds`. CLI invocation is composed of commands, arguments, and flags.

So we are using our own terminology to avoid further homonyms. 

Our Bubble Tea model implementation, our controller, is *Mother*.

Tree leaves (commands that can be invoked interactively or from a script), such as `search`, are *Actions*.

Tree nodes, commands that require further input, such as `admin`, are *Navs*.

## "Global" Variables

A number of development features exist as global singletons driven by static subroutines operating on a single, underlying variable instance.

- `action.go` covers the action map for adding interactive models to Actions.

- A single, shared connection to the Gravwell instance, via the Client library, is serviced by the connection package in `connection.go`.

- `clilog.go` maintains a shared logger for developer logs. It is a shared instance of the gravwell ingest logger.

### Why?

Because the program must be usable from any number of different entry-points and scenarios, it does not have a central "app" struct or similar for hosting widely-shared resources. Cobra and Mother need access to similar resources, without being able to assume who owns or has utilized what.

Similarly, while there are no current plans to implement threading, a singleton is trivial to enforce locks on, especially in software with flexibility in coarseness of locking. 

## Cobra/Bubble Tea Interop

Mother operates on top of an underlying cobra.Command tree, using it for navigation and argument parsing.

Because cobra.Commands cannot support the methods requied to directly interoperate with Bubble Tea, a pre-generated hashtable maps cobra.Commands to their associated Actor interfaces.
Mother keeps track of the active Action (leaf cobra.Command) and looks up its methods in this hashtable. 

```mermaid
flowchart
    subgraph Cobra Command Tree
        root(Nav) <-->  n1(Nav) & n2(Nav)
        n1 <--> n3(Nav) & a1(Action)
        n2 <--> a2(Action) & a3(Action)
        n3 <--> a4(Action) & a5(Action) & a6(Action)
    end
    mother>Mother]
    mother -.*PWD.-> n3
    mother -.*Root.-> root
    mother -.*Action.-> a6
    mother ==*Action==> ActionMap ==*Action's<br>Update()/View()==> mother
```

### Why?

We want to rely on Cobra as much as possible; it has all the navigational features we need and the further we stray from it, the less we benefit from its auto-generation capabilities.

However, Mother cannot hand off control to a cobra.Command leaf (an *Action*) because it does not have `.Update()` and `.View()` methods to supplant her own. We cannot add methods to non-local structs.

With Type Embedding, an Action struct could embed cobra.Command and implement `.Update()` and `.View()` (basically: `class Action extends cobra.Command implements tea.Model` in OOP parlance). That way, it has all the subroutines Cobra will invoke in non-interactive mode and the two we need when driving Bubble Tea.

Solved, right? Not quite. The relationship must be bi-directional, which is not feasible.

Clock this signature `.AddCommand(cmds ...*cobra.Command)`. To get commands into Cobra's tree so it can work its magic, we need to supply a cobra.Command *struct*. Due to the way Go's quasi-inheritance works, we cannot masquerade our Action 'super' type as its 'base'. We can supply cobra with a pointer to the embedded type. ex: 

```go
a := &action.Action{Command: cobra.Command{}}

root.AddCommand(a.Command)
```

This, however, will dispose of our super wrapper `a` as soon as it falls out of scope.

We have a few options:

1) Maintain two, separate-but-topologically-identical trees using two different structures. We retain the normal cobra.Command tree and a parallel tree for Mother to operate on. This decouples Cobra and Mother, allowing them total flexibility in data representation, but could lead to significant data duplication and difficulty guaranteeing equity when adding new commands or performing maintenance. Given Cobra provides all required data for navigation and Nav nodes, this feels a bit like reinventing the wheel just to tack on a couple methods for the tree's leaves.

2) Maintain a data structure of Actions within Mother so we can look up subroutines associated to it when called. This keeps Cobra and Mother paired and allows us to continue leveraging Cobra's tree directly without maintaining a parallel tree. On the other hand, it separates Actions from their subroutines somewhat significantly and would require care to ensure equity, similar to the parallel trees of option #1. 

3) Fork Cobra, attach the required function signatures (ex: `.Update()`, `.View()`, ...) to the Cobra struct directly, and include the fork as a submodule. This is the most straightforward and lowest-initial-lift option. We can navigate and act *entirely* off the cobra.Command tree, supplanting Mother's Model-Update-View with that of the selected Action's stored directly inside the Action's command. However, we now how two packages to maintain, instead of just one.   

While Option 3 seems like the best option right now, future maintainers may not agree, especially as changes occur to the upstream Cobra package. Therefore, option 2 is how interoperability is currently designed. Mother/interactive mode can function entirely off Cobra's navigation and Cobra can operate entirely as normal. The only adaptation takes place in interactive mode, when an action is invoked; Mother uses the action cobra.Command to fetch the methods that should supplant her standard model.

*If you can figure a better adaption pattern, I am all ears.*

## Actions

Actions must satisfy the `action.Model` interface to be able to supplant Mother as the controller. This means satisfying all 5 methods: `Update(), View(), Done(), Reset(), and SetArgs()`.

`Update(tea.Msg) tea.Cmd` is the primary driver of the action. While in handoff mode, Mother will invoke the child's `Update()` subroutine in place of her own.

`View() string`, like Update, supplants Mother's View method while in handoff mode. Note, however, that this is a prompt and all non-interactive output should instead be printed outside of Bubble Tea's control (via `tea.Print*()`).

`Done() bool` is called by mother *before handing off* each cycle. If it is true, Mother will *not* hand off and will instead reassert control and unseat the child. Generally tracked by a private variable in the child struct.

`Reset() error` is called by Mother *after* a child runs, once `Done()` returns true. It resets the child to a clean state so it can be called again later.

`SetArgs([]string) (bool, error)` sets fields in the child that manipulate its next run. It is called when Mother *first enters handoff mode* for a child.

```mermaid
flowchart
    EnterHandoff>Enter<br>Handoff Mode] -->
    SetArgs(child.SetArgs) --> MotherUpdate>Mother.Update] --> Done(child.Done?)
    --false--> Update(child.Update) --> MotherView>Mother.View] -->View[child.View]
    --> Done

    Done --true--> ExitHandoff>Exit<br>Handoff Mode] --> Reset[child.Reset]
```