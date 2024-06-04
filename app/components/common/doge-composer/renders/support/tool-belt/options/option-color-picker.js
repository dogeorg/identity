import {
  LitElement,
  html,
  css
} from "/vendor/@lit/all@3.1.2/lit-all.min.js";

import { defaultOptionStyles } from './styles.js';

class OptionColorPicker extends LitElement {
  static properties = {
    forElement: { type: Object },
    editName: { type: String },
    color: { type: String }
  };

  static styles = [defaultOptionStyles];

  connectedCallback() {
    super.connectedCallback();
    this.addEventListener('click', (e) => e.stopPropagation());
  }

  firstUpdated() {
    this.color = window.getComputedStyle(this.forElement).borderColor; 
  }

  handleColorChange(event) {
    this.forElement.style.borderColor = event.target.getFormattedValue('hex');
  }

  render() {
    return html`
      <sl-color-picker size="small" value=${this.color} @sl-input=${this.handleColorChange}></sl-color-picker>
      <span class="option-text">Border</span>
    `
  }

}

customElements.define("option-color-picker", OptionColorPicker);