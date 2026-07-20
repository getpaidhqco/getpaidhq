import { type RouteConfig, route } from "@react-router/dev/routes";

export default [
  route("/checkout/:slug", "routes/checkout.$slug.tsx"),
] satisfies RouteConfig;
