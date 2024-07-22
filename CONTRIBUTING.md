# Overview

In essence, gwcli is a [Cobra](cobra.dev) tree that can be crawled around via our [Bubble Tea](https://github.com/charmbracelet/bubbletea) instance. As such, you should understand the [Elm Architecture](https://guide.elm-lang.org/architecture/) before continuing. Don't worry, it is really simple. For more on how they work together, see the section below on [Cobra/Bubble Tea Interoperation](#cobrabubble-tea-interop).

gwcli is built to allow more functionality to be easily plugged in. As such, it follows design principles closer to that of a toolbox or framework. For instance, [list scaffolding](utilities/scaffold/list) provides complete functionality for listing any kind of data in a unified way while requiring minimal new code. The goal is to genericize as much as possible, so future developers can simply call these genericized subrotuines.

# Terminology

Bubble Tea has the `tea.Model` interface that must be implemented by a model struct of our own. Bubbles.TextInput, along with every other Bubble, is a tea.Model under the hood. Cobra is composed of `cobra.Commands` and Bubble Tea drives its I/O via `tea.Cmds`. CLI invocation is composed of commands, arguments, and flags.

So we are using our own terminology to avoid further homonyms.

Our Bubble Tea model implementation, our controller, is *Mother*.

Tree leaves (commands that can be invoked interactively or from a script), such as `search`, are *Actions*.

Tree nodes (commands that require further input/are submenus), such as `user`, are *Navs*.

# Quick Tips

- Actions' Update subroutines should **always** return a tea.Cmd when handing control back to Mother.

    - If you do not have anything to tea.Println on completion, use a .Blink method

    - This is to prevent faux-hanging. Bubble Tea only triggers its cycle when a message comes in. Returning nil when few other messages are being sent can cause the application to appear unresponsive when it is instead waiting for another message, thus triggering the anticipated redraw.

- This is a prompt; anything not immediately interactive should be output via tea.Print* as history, rather than in the .View() that will be lost on redraw. 

- Do not include newlines in lipgloss renders. It produces weird results.

# Design and Philosophy

## Command Tree Generation

The command tree is self-building: Each nav knows navs and actions underneath it, 'recurring' downward until only actions are returned.

Root begins generation as it is just a Nav. Take a look at `Execute()` in root.go; you can see that root is given a series of `.New*Nav` and `.New*Action`. Diving into one of the `.New*Nav` subroutines shows that it is built in the same way as root: given a series of self-building Navs and a list of actions that can be invoked at that level.

### Creating a New Nav

To create a new Nav, just create a `.New*Nav` command and add it to the list of nav for the parent nav you want this nav to be accessible under.

The package structure of gwcli reflects the command tree, but there is no real reason this has to be the case.

## "Global" Variables

A number of development features exist as global singletons driven by static subroutines operating on a single, underlying variable instance.

- `action.go` covers the action map for adding interactive models to Actions.

- A single, shared connection to the Gravwell instance, via the Client library, is serviced by the connection package in `connection.go`.

- `clilog.go` maintains a shared logger for developer logs. It is a shared instance of the gravwell ingest logger. See Other Packages for more details on usage style.

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
        root(Nav):::nav <-->  n1(Nav):::nav & n2(Nav):::nav
        n1 <--> n3(Nav):::nav & a1(Action):::action
        n2 <--> a2(Action):::action & a3(Action):::action
        n3 <--> a4(Action):::action & a5(Action):::action & a6(Action):::action
    end
    mother>Mother]
    mother -.*PWD.-> n3
    mother -.*Root.-> root
    mother -.*Action.-> a6
    mother ==*Action==> ActionMap ==*Action's<br>Update()/View()==> mother

    classDef nav stroke:#bb7af7
    classDef action stroke:#f79c7a
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

3) Fork Cobra, attach the required function signatures (ex: `.Update()`, `.View()`, ...) to the Cobra struct directly (or convert the cobra struct to an interface), and include the fork as a submodule. This is the most straightforward and lowest-initial-lift option. We can navigate and act *entirely* off the cobra.Command tree, supplanting Mother's Model-Update-View with that of the selected Action's stored directly inside the Action's command. However, we now how two packages to maintain, instead of just one.

While Option 3 is the most straightforward initially, future maintainers may not agree, especially as changes occur to the upstream Cobra package. Therefore, option 2 is how interoperability is designed. Mother/interactive mode can function entirely off Cobra's navigation and Cobra can operate entirely as normal. The only adaptation takes place in interactive mode, when an action is invoked; Mother uses the action cobra.Command to fetch the methods that should supplant her standard model.

## Actions

Actions must satisfy the `action.Model` interface to be able to supplant Mother as the controller. This means satisfying all 5 methods: `Update(), View(), Done(), Reset(), and SetArgs()`.

`Update(tea.Msg) tea.Cmd` is the primary driver of the action. While in handoff mode, Mother will invoke the child's `Update()` subroutine in place of her own.

`View() string`, like Update, supplants Mother's View method while in handoff mode. Note, however, that this is a prompt and all non-interactive output should instead be printed outside of Bubble Tea's control (via `tea.Print*()`).

`Done() bool` is called by mother *before handing off* each cycle. If it is true, Mother will *not* hand off and will instead reassert control and unseat the child. Generally tracked by a private variable in the child struct.

`Reset() error` is called by Mother *after* `Done()` returns true. It resets the child to a clean state so it can be called again later.

`SetArgs([]string) (string, []tea.Cmd, error)` sets fields in the child that manipulate its next run. It is called when Mother *first enters handoff mode* for a child. It returns, respectively: the reason this argument set is invalid (or ""), tea.Cmds the child needs run on startup (eg: right now), errors outside of the users control. The startup Cmds somewhat take the place of tea.Model.Init().

```mermaid
flowchart
    EnterHandoff>Enter<br>Handoff Mode] -->
    SetArgs(child.SetArgs) --> MotherUpdate>Mother.Update] --> Done(child.Done?)
    --false--> Update(child.Update) --> MotherView>Mother.View] -->View[child.View]
    --> MotherUpdate

    Done --true--> ExitHandoff>Exit<br>Handoff Mode] --> Reset[child.Reset]
```

### Scaffolding

Where possible, use the functionality in the [scaffold](/utilities/scaffold/) package to rapidly construct new actions that fit one of the scaffold archetypes.

## Local Versus Persistent Flags

There are a number of flags that are useful and functionally identically across a number of actions (output, append, CSV/JSON, ...). Therefore, we could make them persistent. However, they do not make sense for some actions, particularly basic actions.

As such, I am not including these common flags as persistent's at root level, lest it require every action to support tangential flags. Instead, common elements of these flags are stored in the stylesheet, to at least provide some degree of consistency across flags that are technically unrelated.

Other flags, such as --script, must be supported by all actions anyways, so they are persistent at a root level. 

# Other Packages

## Supporting Models

[busywait](busywait/busywait.go) and [datascope](datascope/datascope.go) are chameleons: they can operate entirely independently (invoking their own tea.Programs) or be composed into Action structs to be used within Mother.

Busywait is just the spinner bubble, wrapped for consistent appearance.

Datascope is a hybridization of the paginator and viewport bubbles with options specifically for manipulating and fetching search data.

## Stylesheet

Stylesheet provides a shared repository for style constants. The intention is for this to eventually be supplantable for custom styling.

## Logging

Logging is all done via the clilog package, which is just an implementation of the gravwell ingest logger.

Stylistically speaking, callees log relevant data only they have access to, but return errors for the caller to log, lest both callee and caller try to log the same error.