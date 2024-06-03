import {
  LitElement,
  html,
  nothing,
  asyncReplace,
  repeat,
  classMap,
} from "/vendor/@lit/all@3.1.2/lit-all.min.js";

// Utils
import { bindToClass } from "/utils/class-bind.js";

// Lib methods
import * as methods from "./renders/index.js";

// Other components
import * as components from "./renders/support/index.js"


import { styles } from "./styles.js";

class DogeComposer extends LitElement {
  // Declare properties you want the UI to react to changes for.
  static get properties() {
    return {
      actively_editing_container_id: { type: String },
    };
  }

  static styles = styles;

  constructor() {
    super();
    bindToClass(methods, this);
    // Good place to set defaults.
  }

  connectedCallback() {
    super.connectedCallback();
    this.addEventListener('toolbelt-popup-shown', this.handleToolbeltPopupShown)
  }

  handleToolbeltPopupShown(event) {
    this.actively_editing_container_id = event.detail.container_id
  }

  disconnectedCallback() {
    super.disconnectedCallback();
  }

  updated(changedProperties) {
    changedProperties.forEach((oldValue, propName) => {
      console.log(`DOGE-COMPOSER: ${propName} changed. oldValue: ${oldValue}`);
    });
  }

  handleElClick(event, el) {
    console.log(event, el);
  }

  render() {

    return html`
      <div class="elements-container">
        
        ${this.elements.map((el, index) => {
          const containerClasses = {
            'element-container': true,
            'actively-editing': this.actively_editing_container_id === index.toString()
          }
          return html`
          <div class=${classMap(containerClasses)} container_id="${index}">
            ${this[`_render_${el.type}`](el, index)}
          </div>
          <element-divider>
            <sl-icon name="plus-square-fill" label="Add Element"></sl-icon>
          </element-divider>
        `})}
      </div>
    `;
  }
}

customElements.define("doge-composer", DogeComposer);