// @ts-check
import { defineConfig } from "astro/config";
import starlight from "@astrojs/starlight";

// https://astro.build/config
export default defineConfig({
  integrations: [
    starlight({
      title: "vel",
      social: [
        {
          icon: "github",
          label: "GitHub",
          href: "https://github.com/dennypenta/vel",
        },
      ],
      sidebar: [
        {
          label: "Guides",
          autogenerate: { directory: "guides" },
        },
        // {
        //   label: "Tutorials",
        //   autogenerate: { directory: "tutorials" },
        // },
        // {
        //   label: "Reference",
        //   autogenerate: { directory: "reference" },
        // },
      ],
      customCss: ["./src/styles/theme.css"],
    }),
  ],
});
