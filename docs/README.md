# javaup documentation site

The documentation site uses Hugo and the `hugo-geekdoc` theme. English is
served at the site root and Simplified Chinese under `/zh-cn/`.

## Local preview

Install Hugo 0.160.0 or newer, then download the pinned prebuilt theme and
start the development server:

```shell
bash docs/scripts/setup-theme.sh
hugo server --source docs
```

On Windows, use PowerShell for the theme setup:

```powershell
./docs/scripts/setup-theme.ps1
hugo server --source docs
```

Build the production site with:

```shell
hugo --source docs --gc
```

Generated files and the downloaded theme are intentionally ignored by Git.
The theme version and archive checksum are pinned in `docs/THEME_VERSION` and
`docs/THEME_SHA256`. Pushes to `main` are built and published by the `docs.yml`
GitHub Actions workflow.
