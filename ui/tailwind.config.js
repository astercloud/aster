/** @type {import('tailwindcss').Config} */
export default {
  content: ["./index.html", "./src/**/*.{vue,js,ts,jsx,tsx}"],
  darkMode: "class",
  theme: {
    extend: {
      colors: {
        primary: {
          DEFAULT: "#3b82f6",
          hover: "#2563eb",
          light: "#dbeafe",
          dark: "#1e40af",
        },
        surface: {
          DEFAULT: "#ffffff",
          dark: "#334155", // 更温和的深色表面 slate-800
        },
        background: {
          DEFAULT: "#fafafa",
          dark: "#1e293b", // 更明亮的深色背景 slate-900
        },
        border: {
          DEFAULT: "#e5e7eb",
          dark: "#475569", // 调整边框色为 slate-600
        },
        secondary: {
          DEFAULT: "#6b7280",
          dark: "#cbd5e1", // 提高次要文本的对比度 slate-300
        },
        text: {
          DEFAULT: "#111827",
          dark: "#f1f5f9", // 更明亮的文本色 slate-100
        },
      },
      animation: {
        "slide-in": "slideIn 0.3s ease-out",
        "slide-up": "slideUp 0.3s ease-out",
        "fade-in": "fadeIn 0.2s ease-out",
        "pulse-slow": "pulse 3s cubic-bezier(0.4, 0, 0.6, 1) infinite",
        "bounce-subtle": "bounceSubtle 2s infinite",
      },
      keyframes: {
        slideIn: {
          "0%": { opacity: "0", transform: "translateY(10px)" },
          "100%": { opacity: "1", transform: "translateY(0)" },
        },
        slideUp: {
          "0%": { opacity: "0", transform: "translateY(20px)" },
          "100%": { opacity: "1", transform: "translateY(0)" },
        },
        fadeIn: {
          "0%": { opacity: "0" },
          "100%": { opacity: "1" },
        },
        bounceSubtle: {
          "0%, 100%": { transform: "translateY(0)" },
          "50%": { transform: "translateY(-5px)" },
        },
      },
      screens: {
        xs: "475px",
      },
    },
  },
  plugins: [],
};
