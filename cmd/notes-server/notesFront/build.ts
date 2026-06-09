// Build script for notes-front
import { join } from "path";

const isDev = process.argv.includes("--dev");

// Clean and prepare dist/
try {
  await Bun.$`rm -rf dist`;
} catch (e) {}
await Bun.$`mkdir -p dist`;

console.log(`Building frontend in ${isDev ? "Development" : "Production"} mode...`);

// Run Bun bundler
const result = await Bun.build({
  entrypoints: ["./src/main.ts"],
  outdir: "./dist",
  minify: !isDev,
  sourcemap: isDev ? "external" : "none",
});

if (!result.success) {
  console.error("❌ Build failed:");
  for (const message of result.logs) {
    console.error(message);
  }
  process.exit(1);
}

// Copy index.html to dist/
const indexHtml = Bun.file("./src/index.html");
await Bun.write("./dist/index.html", indexHtml);

console.log("✅ Build completed successfully!");
if (isDev) {
  console.log("Watching for changes...");
}
