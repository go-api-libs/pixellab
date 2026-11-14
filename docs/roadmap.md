# Roadmap

## Descriptions the library drops

Of upstream's descriptions, some reach no doc comment.

- **What compress merges.** Of two near-duplicate schemas, compress keeps one description: most of the
  "Response model for ..." of the bodies `bodyNames` names, and `StyleOptions`' "Copy ... from style image" beside
  `ProFlashStyleOptions`. A merged schema could keep each member's description on the properties that referred to
  it, as `consolidateEnums` does.
- **Defaults compress merges.** Of two enums alike but for their default, compress keeps one default: the
  `CreateMapObjectRequest.view` upstream defaults to `high top-down` says `low top-down`. compress could move
  each one's default beside the `$ref`s that pointed at it, as it does descriptions that disagree.
