import type { Metadata, Viewport } from "next";
import { Archivo, JetBrains_Mono, Public_Sans } from "next/font/google";
import "./globals.css";

const display = Archivo({
  variable: "--ff-display",
  subsets: ["latin"],
  axes: ["wdth"],
  display: "swap",
});

const body = Public_Sans({
  variable: "--ff-body",
  subsets: ["latin"],
  display: "swap",
});

const data = JetBrains_Mono({
  variable: "--ff-data",
  subsets: ["latin"],
  display: "swap",
});

export const metadata: Metadata = {
  title: { default: "Terminus", template: "%s · Terminus" },
  description: "Quanto tempo sua equipe fica parada em cada ponto do roteiro.",
};

export const viewport: Viewport = {
  themeColor: [
    { media: "(prefers-color-scheme: light)", color: "#ECEFF4" },
    { media: "(prefers-color-scheme: dark)", color: "#181818" },
  ],
};

// Applies a saved theme choice before paint (no flash).
const themeScript = `try{var t=localStorage.getItem("terminus-theme");if(t==="light"||t==="dark")document.documentElement.dataset.theme=t}catch(e){}`;

export default function RootLayout({ children }: LayoutProps<"/">) {
  return (
    <html
      lang="pt-BR"
      suppressHydrationWarning
      className={`${display.variable} ${body.variable} ${data.variable} h-full antialiased`}
    >
      <head>
        <script dangerouslySetInnerHTML={{ __html: themeScript }} />
      </head>
      <body className="min-h-full">{children}</body>
    </html>
  );
}
