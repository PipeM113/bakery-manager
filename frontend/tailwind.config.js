// Brand palette and fonts of the manual (angeles_bakery_web/docs/diseno.md); see docs/diseno.md.
// Salvia and rosa are backgrounds only; gold text on light grounds is always "oro";
// "oro-claro" is for dark grounds only.
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
        crema: "#FAF6F1",
        lino: "#E8DDD3",
        salvia: "#C4C9B8",
        rosa: "#D9B8AE",
        negro: "#1C1C1C",
        oro: "#735523",
        "oro-claro": "#EECC6A",
        peligro: "#8A1C1C",
        linea: "rgba(115, 85, 35, 0.35)",
      },
      fontFamily: {
        display: ["Playfair Display", "Iowan Old Style", "Georgia", "serif"],
        sans:    ["DM Sans", "system-ui", "-apple-system", "Segoe UI", "sans-serif"],
      },
    },
  },
  plugins: [],
}
