import express from "express";
import { fileURLToPath } from "node:url";
import path from "node:path";

const __dirname = path.dirname(fileURLToPath(import.meta.url));

/**
 * Builds the Express application.
 *
 * State is kept in-memory and seeded per instance so the app is trivially
 * runnable in the Cloud Agent environment without an external datastore.
 */
export function createApp() {
  const app = express();
  app.use(express.json());

  let nextId = 1;
  const explorations = [];

  const seed = (title, url) => {
    explorations.push({ id: nextId++, title, url, createdAt: new Date().toISOString() });
  };
  seed("Express documentation", "https://expressjs.com");
  seed("Node.js docs", "https://nodejs.org/en/docs");

  app.get("/api/health", (_req, res) => {
    res.json({ status: "ok", uptime: process.uptime() });
  });

  app.get("/api/explorations", (_req, res) => {
    res.json({ explorations });
  });

  app.post("/api/explorations", (req, res) => {
    const { title, url } = req.body ?? {};
    if (typeof title !== "string" || title.trim() === "") {
      return res.status(400).json({ error: "title is required" });
    }
    if (typeof url !== "string" || url.trim() === "") {
      return res.status(400).json({ error: "url is required" });
    }
    const exploration = {
      id: nextId++,
      title: title.trim(),
      url: url.trim(),
      createdAt: new Date().toISOString(),
    };
    explorations.push(exploration);
    res.status(201).json(exploration);
  });

  app.use(express.static(path.join(__dirname, "..", "public")));

  return app;
}
