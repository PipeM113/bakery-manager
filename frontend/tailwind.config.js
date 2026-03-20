export default {
  content: ["./index.html", "./src/**/*.{js,ts,jsx,tsx}"],
  theme: {
    extend: {
      colors: {
        noir:   { DEFAULT: "#0A0A0A", 800: "#141414", 700: "#1C1C1C", 600: "#2A2A2A" },
        gold:   { DEFAULT: "#C9A84C", light: "#E2C97E", dark: "#A67C2E" },
        cream:  { DEFAULT: "#F5EDD6", muted: "#D4C9A8" },
      },
      fontFamily: {
        display: ["Cormorant Garamond", "serif"],
        sans:    ["Jost", "sans-serif"],
      },
    },
  },
  plugins: [],
}