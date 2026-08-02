# javaup documentation site

The documentation site uses Astro and the official Starlight documentation
theme. English is served at the site root and Simplified Chinese under
`/zh-cn/`.

## Local development

Install Node.js 22.12 or newer, then run:

```shell
cd docs
npm install
npm run dev
```

Create a production build with:

```shell
npm run build
```

The generated site is written to `docs/dist`. Pushes to `main` are built and
published to GitHub Pages by the `docs.yml` workflow.
