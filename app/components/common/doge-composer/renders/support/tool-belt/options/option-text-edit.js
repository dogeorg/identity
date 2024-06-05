import { LitElement, html, css } from "/vendor/@lit/all@3.1.2/lit-all.min.js";

import { defaultOptionStyles } from "./styles.js";

class OptionTextEdit extends LitElement {
  static properties = {
    forElement: { type: Object },
    editName: { type: String },
    textValue: { type: String },
    optionLabel: { type: String },
    elementPropertyName: { type: String },
  };

  static styles = [defaultOptionStyles];

  connectedCallback() {
    super.connectedCallback();
    this.addEventListener("click", (e) => this.handleOptionClick(e));
  }

  disconnectedCallback() {
    this.removeEventListener("click", (e) => this.handleOptionClick(e));
    super.disconnectedCallback();
  }

  firstUpdated() {
    this.textValue = this.forElement.textContent;
    this.forElement.addEventListener("focus", (e) => this.handleFocus(e));
    this.forElement.addEventListener("blur", (e) => this.handleBlur(e));
    this.forElement.addEventListener("input", (e) => this.handleInput(e));
  }

  handleOptionClick() {
    const isEditing = this.forElement.getAttribute("contenteditable") === "true";
    this.forElement.setAttribute("contenteditable", !isEditing);
    if (!isEditing) {
      this.forElement.focus();
    }
  }

  handleFocus() {
    this.forElement.classList.add("is-focused");
  }

  handleBlur() {
    this.forElement.classList.remove("is-focused");
  }

  handleInput() {
    // Dispatch a custom event indicating textField change.
    const textFieldChangeEvent = new CustomEvent('value-change', {
      detail: {
        forElement: this.forElement,
        editName: this.editName,
      },
      bubbles: true,
      composed: true
    });
    this.dispatchEvent(textFieldChangeEvent);
  }

  render() {
    return html`
      <sl-icon name="input-cursor-text"></sl-icon>
      <span class="option-text">${this.optionLabel || "?!"}</span>
    `;
  }
}

customElements.define("option-text-edit", OptionTextEdit);
