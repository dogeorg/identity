import { Router } from "/vendor/@vaadin/router@1.7.5/vaadin-router.min.js";
import { doFoo } from "./middleware.js";

let router;

export const getRouter = (targetElement) => {
  if (!router) {
    router = new Router(targetElement);

    // More Advanced Route table.  Cached Components.
    router.setRoutes([
      {
        path: "/",
        component: "profile-view",
        action: async (context, commands) => {
          // Render component (ideally an existing)
          const existing = targetElement.querySelector("profile-view");
          if (existing) return existing;
          return commands.component("profile-view");
        },
      },
      {
        path: "/edit",
        component: "profile-edit-view",
        action: async (context, commands) => {
          // Render component (ideally an existing)
          const existing = targetElement.querySelector("profile-edit-view");
          if (existing) return existing;
          return commands.component("profile-edit-view");
        },
      },
      {
        path: "/foo/:id*",
        action: async (context, commands) => {
          // Ensure the profile middleware is processed
          await doFoo(context, commands);

          // Render component (ideally an existing)
          const existing = targetElement.querySelector("profile-view");
          if (existing) return existing;
          return commands.component("profile-view");
        },
      },
    ]);
  }
  return router;
};
