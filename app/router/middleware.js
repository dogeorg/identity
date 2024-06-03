import { store } from "/state/store.js";

// Example middleware
export function logPathMiddleware(context, commands) {
  console.log("Navigating to path:", context.pathname);
  return undefined; // Proceed with the navigation
}

export async function doFoo(context, commands) {
  console.log("Running showProfile middleware ..", context.params.id);

  // Set the "inspectedFoo" as the first param after "foo" in the URL
  // Eg. localhost:9090/foo/bar, will set "bar" as the inspected id

  store.updateState({
    identityContext: {
      inspectedFoo: context.params.id[0],
    },
  });

  // Return undefined so that routing is not interrupted.
  // The navigation should hapilly continue to its detination.
  return undefined;
}
