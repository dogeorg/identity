import {
  LitElement,
  html,
  css
} from "/vendor/@lit/all@3.1.2/lit-all.min.js";

class ElementDivider extends LitElement {
  // Declare properties you want the UI to react to changes for.
  static get properties() {
    return {};
  }

  static styles = css`
    :host {
      display: block;
      margin: 5px 0;
      padding: 0;
    }
    ::slotted(sl-icon) {
      font-size: 0rem;
      transition: font-size 100ms ease-out;
    }

    .divider-container:hover {
      gap: 1em;
      color: yellow;
      cursor: pointer;
    }

    .divider-container:hover ::slotted(sl-icon) {
      font-size: 1.5rem;
    }

    .divider-container {
      display: flex;
      flex-direction: row;
      align-items: center;
      justify-content: center;
      gap: 0em;
      width: 100%;
      box-sizing: border-box;
      padding: 10px 0px;

      transition: font-size 100ms ease-in;
    }

    .divide {
      display: flex;
      width: 40px;
      border-bottom: 4px dotted #999;
    }
  `

  constructor() {
    super();
  }

  connectedCallback() {
    super.connectedCallback();
  }

  disconnectedCallback() {
    super.disconnectedCallback();
  }

  render() {

    return html`
      <div class="divider-container">
        <div class="divide"></div>
        <slot></slot>
        <div class="divide"></div>
      </div>
    `;
  }
}

customElements.define("element-divider", ElementDivider);