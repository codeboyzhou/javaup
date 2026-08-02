import { unified } from '@astrojs/markdown-remark';
import { defineConfig } from 'astro/config';
import mermaid from 'astro-mermaid';
import starlight from '@astrojs/starlight';
import rehypeCjkSpacing from './plugins/rehype-cjk-spacing.mjs';

export default defineConfig({
  site: 'https://codeboyzhou.github.io',
  base: '/javaup',
  publicDir: './static',
  markdown: {
    processor: unified({ rehypePlugins: [rehypeCjkSpacing] }),
  },
  integrations: [
    mermaid({ enableLog: false }),
    starlight({
      title: {
        en: 'javaup documentation',
        'zh-CN': 'javaup 项目文档',
      },
      description: 'Project-aware Java toolchains for Maven builds.',
      defaultLocale: 'root',
      locales: {
        root: { label: 'English', lang: 'en' },
        'zh-cn': { label: '简体中文', lang: 'zh-CN' },
      },
      social: [
        {
          icon: 'github',
          label: 'GitHub',
          href: 'https://github.com/codeboyzhou/javaup',
        },
      ],
      editLink: {
        baseUrl: 'https://github.com/codeboyzhou/javaup/edit/main/docs/',
      },
      lastUpdated: true,
      customCss: ['./src/styles/custom.css'],
      sidebar: [
        {
          label: 'Start Here',
          translations: { 'zh-CN': '快速上手' },
          items: [
            { slug: 'start-here/installation' },
            { slug: 'start-here/quick-start' },
          ],
        },
        {
          label: 'User Guide',
          translations: { 'zh-CN': '使用指南' },
          items: [
            { slug: 'user-guide/managing-projects' },
            { slug: 'user-guide/running-maven' },
            { slug: 'user-guide/maven-settings' },
            { slug: 'user-guide/updates-and-uninstallation' },
          ],
        },
        {
          label: 'Reference',
          translations: { 'zh-CN': '参考' },
          items: [
            { slug: 'reference/command-reference' },
            { slug: 'reference/detection-rules' },
            { slug: 'reference/configuration-and-storage' },
            { slug: 'reference/troubleshooting' },
            { slug: 'reference/project-scope' },
          ],
        },
      ],
    }),
  ],
});
