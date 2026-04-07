// @ts-check
import { defineConfig } from 'astro/config';
import tailwindcss from "@tailwindcss/vite";

import cloudflare from "@astrojs/cloudflare";

// https://astro.build/config
export default defineConfig({
  site: process.env.PUBLIC_SITE_URL || undefined,
  session: {
    driver: {
      entrypoint: "unstorage/drivers/null",
    },
  },
  vite: {
      plugins: [tailwindcss()],
    },

  adapter: cloudflare({
    imageService: "compile",
  }),
});