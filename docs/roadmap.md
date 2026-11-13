# Roadmap

## Descriptions the library drops

Of upstream's descriptions, some reach no doc comment:

- **An image size's own rules.** `consolidateImageSizes` writes each request's bounds and step beside its
  `ImageSize`, and drops the descriptions of the width and height it merged. Some say what no bound does: Pixen's
  "max area 512x512", "must equal height when either side is below 32", and a UI asset's "max per axis depends on
  aspect". They belong in the property's description, where the bounds are.
- **What compress merges.** Of two near-duplicate schemas, compress keeps one description: most of the
  "Response model for ..." of the bodies `bodyNames` names, and `StyleOptions`' "Copy ... from style image" beside
  `ProFlashStyleOptions`. A merged schema could keep each member's description on the properties that referred to
  it, as `consolidateEnums` does.
