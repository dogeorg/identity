import {
  LitElement,
  html,
  css
} from "/vendor/@lit/all@3.1.2/lit-all.min.js";

import { bindToClass } from "/utils/class-bind.js";
import * as methods from "./lib/index.js";

class Toolbelt extends LitElement {
  static properties = {
    forElement: { type: Object },
    editName: { type: String },
    active: { type: Boolean },
  };

  static styles = css`
    :host {
      position: relative;
      display: block;
    }

    sl-popup {
      --arrow-color: #f026db;
      --arrow-size: 0.7rem;
    }

    sl-popup::part(popup) {
      box-shadow: 2px 2px 4px rgba(0,0,0,0.2);
    }

    .box {
      display: flex;
      justify-content: center;
      align-items: start;
      background: #f026db;
      border-radius: var(--sl-border-radius-medium);
    }

    .options {
      display: flex;
      flex-direction: row;
      align-items: center;
      justify-content: center;
      gap: 4px;
      padding: 4px;
    }

    .option {
      display: flex;
      flex-direction: column;
      gap: 4px;
      align-items: center;
      justify-content: center;
      width: 40px;
      height: 50px;
      padding: 8px;
      padding-bottom: 4px;
      color: black;
      user-select: none;
    }

    .option sl-icon {
      font-size: 1.35rem;
    }

    .option .option-text {
      font-size: 0.7rem;
      text-transform: uppercase;
      font-weight: bold;
      user-select: none;
    }

    .option:hover {
      cursor: pointer;
      background: rgba(255,255,255, 0.25);
    }
  `;

  constructor() {
    super();
    bindToClass(methods, this);
    this.forElement = null;
    this.editName = '';
  }

  connectedCallback() {
    super.connectedCallback();
  }

  show() {
      this.active = true;
    }

  hide() {
    this.active = false;
  }

  render() {
    return html`
      <sl-popup
        ?active=${this.active}
        placement="bottom"
        arrow
        arrow-placement="anchor"
        distance="2"
        .anchor=${this.forElement}
      >
        <div class="box">
          <div class="options">
            ${this.generateOptions(this.forElement)}
          </div>
          <div class="drawer"></div>
        </div>
      </sl-popup>
    `;
  }
}

customElements.define("tool-belt", Toolbelt);