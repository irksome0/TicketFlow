import type { Config } from "tailwindcss";

const config: Config = {
  content: [
    "./app/**/*.{js,ts,jsx,tsx,mdx}",
    "./components/**/*.{js,ts,jsx,tsx,mdx}",
    "./lib/**/*.{js,ts,jsx,tsx,mdx}",
  ],
  theme: {
    extend: {
      colors: {
        border: "#d9e2ec",
        surface: "#f7fafc",
        text: "#17202a",
        muted: "#627386",
        primary: "#155e75",
        danger: "#b42318",
        success: "#027a48",
        warning: "#b54708",
      },
    },
  },
  plugins: [],
};

export default config;
