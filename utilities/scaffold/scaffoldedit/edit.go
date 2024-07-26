package scaffoldedit

/**
 * More on Design:
 * Edit is definitely the most complex of the scaffolds, requiring components of both Create
 * (arbitrary TIs) and Delete (list possible structs/items).
 * By virtue of passing around structs and ids, it was always going to require multiple generics.
 * However, the decision to not use reflection was made fairly early.
 * I figured that reflection is
 * 1) slow
 * 2) error-prone (needing to look up qualified field names given by the implementor)
 * 3) an added layer of complexity on top of the already-in-play generics
 * Thus, no reflection.
 * The side effect of this, of course, is that we need yet more functions from the implementor and a
 * couple of trivial get/sets to be able to operate on the struct we want to update.
 *
 * Not sharing the Field struct between edit and create was a conscious choice to allow them to be
 * updated independently as it is more coincidental that they are similar.
 */
