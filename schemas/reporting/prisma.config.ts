import "dotenv/config";
import { defineConfig, env } from "prisma/config";

export default defineConfig({
  schema: "schema.prisma",
  datasource: {
    url: env("REPORTING_DATABASE_URL"),
  },
});
