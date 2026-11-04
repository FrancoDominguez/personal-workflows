# Repo structure

The decided layout for this repo. New code goes where this says it goes.

```
app/                    Go module (francodominguez.com/gmail-webhook/app)
  function.go           root stub — blank-imports each function package below
  functions/
    unsub/              one folder per Cloud Run function, package named after it
      unsub.go          handler + functions.HTTP registration
      envelope.go       per-function helpers: separate files, same package, unexported
  utils/                package utils — code shared by more than one function
  cmd/                  regular binaries (package main): local server, oauth quickstart,
                        and any future workload that is NOT a function (e.g. VM jobs)
terrafrom/              all infrastructure
docs/                   decisions and notes like this one
```

## Rules

- **One folder per function** under `functions/`. The folder is its own package;
  the handler file carries the function's name, helpers live in sibling files in
  the same package, unexported. Other functions cannot reach them — that privacy
  is the point.
- **Adding a function** = new folder under `functions/` + one blank-import line in
  `app/function.go`. The root stub exists because the GCF buildpack only imports
  the module root (`pack build --path ./app` in CI); a function whose `init()`
  isn't reachable from the root never registers.
- **A helper graduates to `utils/`** only when a second function actually needs
  it — not when it merely looks reusable.
- **Anything with a `main()`** goes under `cmd/<name>/`. These are plain binaries:
  they may import `utils/`, but the buildpack ignores them and they don't register
  functions.
- Growing deeper is the same pattern again: a function folder that gets big can
  split further (e.g. `functions/unsub/internal/…`), since `internal/` under the
  function's folder is only importable within it.
