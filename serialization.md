## Type metadata

Go resolves tag names from the built-in type table in `marshaler.go`.
Custom types may provide:

```go
TmarkTagName() string
```

Go resolves line mode from the built-in type table in `marshaler.go`.
Custom types may provide:

```go
TmarkLayout() ExpandMode
```

`ExpandMode` values map to the language-neutral line modes:

| Go value | spec line mode |
|---|---|
| `NoExpand` | `none` |
| `Expand` | `always` |
| `ExpandMultiple` | `multiple` |

## Field tags

Go struct fields use the `tmark` tag:

| Go tag | spec rule |
|---|---|
| `tmark:"named"` | named field, using lowercase Go field name |
| `tmark:"named,name"` | named field, using `name` |
| `tmark:"unnamed"` | unnamed field |

## Unknown tags

`Unmarshal` rejects unknown node tags by default.

Use `DefaultUnmarshaler.Soft().Unmarshal(data, &v)` to preserve unknown valid
nodes as `Unknown`. `Unknown` can be used in both block and inline positions and
keeps the raw node bytes, including the opening and closing braces.
