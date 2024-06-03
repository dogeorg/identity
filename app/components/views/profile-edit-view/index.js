import {
  LitElement,
  html,
  nothing,
  asyncReplace,
  repeat,
} from "/vendor/@lit/all@3.1.2/lit-all.min.js";
import { store } from "/state/store.js";
import { StoreSubscriber } from "/state/subscribe.js";

// APIs
import { getIdentity } from "/api/identity/identity.js";

// Utils
import { bindToClass } from "/utils/class-bind.js";

// Lib methods
// import * as classMethods from "./lib/index.js";

// Other components
import * as components from '/components/common/doge-composer/index.js';

import { styles } from "./styles.js";

// Dummy data
import { PROFILE_COMPOSITION } from "./fixtures/profile-composition.js";

class ProfileEditView extends LitElement {
  // Declare properties you want the UI to react to changes for.
  static get properties() {
    return {
      identity: { type: Object },
      profile_composition: { type: Object },
    };
  }

  static styles = styles;

  constructor() {
    super();
    // bindToClass(classMethods, this);
    // Good place to set defaults.
    this.identity;
    this.profile_composition = PROFILE_COMPOSITION
  }

  connectedCallback() {
    super.connectedCallback();
    this.context = new StoreSubscriber(this, store);
    this.fetchData();
  }

  disconnectedCallback() {
    super.disconnectedCallback();
  }

  updated(changedProperties) {
    changedProperties.forEach((oldValue, propName) => {
      console.log(`PROFILE-EDIT-VIEW: ${propName} changed. oldValue: ${oldValue}`);
    });
  }

  async fetchData() {
    this.identity = await getIdentity();
  }

  render() {
    const { identityContext } = this.context.store;

    return html`
      <div>
        <h3>Edit Profile</h3>
        <doge-composer
          .elements=${this.profile_composition}
        >
        </doge-composer>
      </div>
    `;
  }
}

customElements.define("profile-edit-view", ProfileEditView);