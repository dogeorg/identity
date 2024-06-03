import ApiClient from "/api/client.js";
import { store } from "/state/store.js";

import { generateMockedIdentity } from "./identity.mocks.js";

const client = new ApiClient("http://localhost:3000", store.networkContext);

export async function getIdentity() {
  return client.get("/identity", { mock: generateMockedIdentity });
}
