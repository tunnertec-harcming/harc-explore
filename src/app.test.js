import { test } from "node:test";
import assert from "node:assert/strict";
import request from "supertest";
import { createApp } from "./app.js";

test("GET /api/health reports ok", async () => {
  const app = createApp();
  const res = await request(app).get("/api/health");
  assert.equal(res.status, 200);
  assert.equal(res.body.status, "ok");
});

test("GET /api/explorations returns seeded data", async () => {
  const app = createApp();
  const res = await request(app).get("/api/explorations");
  assert.equal(res.status, 200);
  assert.ok(Array.isArray(res.body.explorations));
  assert.ok(res.body.explorations.length >= 2);
});

test("POST /api/explorations creates a record", async () => {
  const app = createApp();
  const res = await request(app)
    .post("/api/explorations")
    .send({ title: "Cursor", url: "https://cursor.com" });
  assert.equal(res.status, 201);
  assert.equal(res.body.title, "Cursor");
  assert.ok(res.body.id);

  const list = await request(app).get("/api/explorations");
  const titles = list.body.explorations.map((e) => e.title);
  assert.ok(titles.includes("Cursor"));
});

test("POST /api/explorations rejects missing fields", async () => {
  const app = createApp();
  const res = await request(app).post("/api/explorations").send({ title: "" });
  assert.equal(res.status, 400);
  assert.ok(res.body.error);
});
