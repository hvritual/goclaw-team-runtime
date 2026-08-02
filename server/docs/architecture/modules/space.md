# Space module

- Public seam: internal/modules/space/contract
- Implementation: internal/modules/space/internal
- Proto contracts: api/space/v1/space.proto and api/space/v1/asset.proto
- SQLite provider: internal/modules/space/internal/infrastructure/sqlite
- Custom HTTP adapters: internal/modules/space/interfaces/http

Space owns Asset identity, immutable content-version facts, object keys,
checksums, sizes and upload/cleanup lifecycle. It does not own why an Asset is
attached to an Issue, comment, Project, Knowledge entry or avatar. Consumers
store only their relation facts and call the Space contract for Asset behavior.

The SQLite provider owns `space_assets`, `space_asset_versions` and
`space_upload_intents`. Consumer relation tables, including
`issue_asset_refs`, remain outside the provider. Cross-context foreign keys and
cascades are forbidden; deletion and reclamation must be explicit application
workflows.

`WorkspaceAccess` is supplied by Auth composition. Space application code does
not read Auth or Workspace tables. Object storage is supplied through
`contract.ObjectStore`; local/S3 details remain outside domain/application.

Workspace reads and downloads always pass through `AssetService` authorization.
The `/uploads/*` compatibility adapter serves only generated personal direct
objects used by avatars and rejects all workspace keys.

After Proto changes, run `make generate`. Register provider and contract edges
in `internal/bootstrap/application.go` and keep custom multipart translation in
the interface layer.
