/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ["./public/**/*.html", "./admin/**/*.html"],
  darkMode: "class",
  theme: {
    extend: {
      colors: {
        "primary": "#22c55e",
        "primary-blue": "#137fec",
        "logo-green": "#22c55e",
        "logo-red": "#ef4444",
        "logo-black": "#1e293b",
        "logo-blue": "#1e40af",
        "background-light": "#f6f7f8",
        "background-dark": "#101922",
      },
      fontFamily: { display: ["Inter"] },
      borderRadius: { DEFAULT: "0.25rem", lg: "0.5rem", xl: "0.75rem", full: "9999px" },
    },
  },
  plugins: [],
}
