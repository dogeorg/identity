import { LitElement, html, css } from "/vendor/@lit/all@3.1.2/lit-all.min.js";

import { defaultOptionStyles } from "./styles.js";

class OptionColorPicker extends LitElement {
  static properties = {
    forElement: { type: Object },
    editName: { type: String },
    color: { type: String },
    optionLabel: { type: String },
    elementPropertyName: { type: String },
    allowOpacity: { type: Boolean },
  };

  static styles = [defaultOptionStyles];

  connectedCallback() {
    super.connectedCallback();
    this.addEventListener("click", (e) => e.stopPropagation());
  }

  firstUpdated() {
    this.color = window.getComputedStyle(this.forElement)[
      this.elementPropertyName
    ];
  }

  handleColorChange(event) {
    this.forElement.style[this.elementPropertyName] =
      event.target.getFormattedValue("rgba");
  }

  render() {
    return html`
      <sl-color-picker
        size="small"
        value=${this.color}
        @sl-input=${this.handleColorChange}
        ?opacity=${this.allowOpacity}
      ></sl-color-picker>
      <span class="option-text">${this.optionLabel || "?!"}</span>
    `;
  }
}

customElements.define("option-color-picker", OptionColorPicker);

