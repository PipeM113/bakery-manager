export default {
  content: ["./index.html", "./src/**/*.{js,ts,jsx,tsx}"],
  theme: {
    extend: {
      fontSize: {
        xs:   ["13px", "1.5"],
        sm:   ["15px", "1.6"],
        base: ["17px", "1.7"],
        lg:   ["19px", "1.7"],
        xl:   ["21px", "1.8"],
      },
      colors: {
        vanilla: { DEFAULT: "#FDFAF4", 100: "#F7F0E4", 200: "#EDE0C8" },
        terracota: { 400: "#C4714A", 500: "#B05A35", 600: "#8E4528" },
        olive:   { 500: "#5C6B3A", 600: "#47542C" },
        stone:   { 100: "#F5F0E8", 200: "#E8DFD0", 400: "#5C5248", 600: "#4A3D35", 800: "#1E1410" },
        gold:    { DEFAULT: "#B8860B", light: "#D4A017", dark: "#8B6508" },
      },
      fontFamily: {
        display: ["Cormorant Garamond", "serif"],
        sans:    ["Jost", "sans-serif"],
      },
    },
  },
  plugins: [],
}