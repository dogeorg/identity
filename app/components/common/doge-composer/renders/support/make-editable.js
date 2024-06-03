import {
  LitElement,
  html,
  css
} from "/vendor/@lit/all@3.1.2/lit-all.min.js";

import * as toolbelt from "./editable-toolbelt.js";

class MakeEditable extends LitElement {
  static get properties() {
    return {
      container_id: { type: String }
    };
  }

  static styles = css`
    .toolbelt {
      position: relative;
    }
    .toolbelt sl-popup {
      --arrow-color: #ff4cec;
    }

    .toolbelt .box {
      width: 100px;
      height: 40px;
      display: flex;
      justify-content: center;
      align-items: center;
      background: #ff4cec;
      border-radius: var(--sl-border-radius-medium);
    }
  `;

  constructor() {
    super();
  }

  connectedCallback() {
    super.connectedCallback();
    this.initEditableFields();

    // Listen for clicks outside of editable fields to close the popup
    this.addEventListener('click', (event) => this.closeToolbelt(event));
  }

  initEditableFields() {
    // Wait for the slot's contents to be initialized
    this.shadowRoot.addEventListener('slotchange', e => {
      const slot = e.target;
      const nodes = slot.assignedElements({flatten: true});
      nodes.forEach(node => {
        this.attachEditListeners(node);
      });
    });
  }

  attachEditListeners(node) {
    const editableElements = node.shadowRoot.querySelectorAll('[data-edit-type]');
    editableElements.forEach((element, index) => {
      element.addEventListener('click', (event) => this.showToolbelt(event, element, index));
    });
  }

  showToolbelt(event, element, elementIndex) {
    // Unique identifier for the toolbelt
    const toolbeltId = `${elementIndex}_${element.tagName}`;

    // Attempt to find an existing toolbelt for this element
    let toolbelt = this.shadowRoot.querySelector(`editable-toolbelt[id="${toolbeltId}"]`);

    // If no existing toolbelt, create a new one
    if (!toolbelt) {
        toolbelt = document.createElement('editable-toolbelt');
        toolbelt.id = toolbeltId; // Set the unique ID for reference
        toolbelt.forElement = element;
        toolbelt.editName = element.getAttribute('data-edit-name');
        this.shadowRoot.appendChild(toolbelt);
    }

    // Hide all toolbelts
    this.shadowRoot.querySelectorAll('editable-toolbelt').forEach(tb => {
      tb.hide(); 
    });

    // Show the relevant toolbelt
    toolbelt.show();
    event.stopPropagation(); // Prevent the click from closing the toolbelt immediately

    // Dispatch a custom event with details about the popup being shown
    const popupShownEvent = new CustomEvent('toolbelt-popup-shown', {
      detail: {
          toolbeltId: toolbeltId,
          container_id: this.container_id,
      },
      bubbles: true,
      composed: true
    });
    this.dispatchEvent(popupShownEvent);
  }

  closeToolbelt(event) {
    const toolbelts = this.shadowRoot.querySelectorAll('editable-toolbelt');
    toolbelts.forEach(toolbelt => {
      if (!toolbelt.contains(event.target)) {
        toolbelt.hide();
      }
    });
  }

  render() {
    return html`
      <slot></slot>
    `;
  }

  disconnectedCallback() {
    super.disconnectedCallback();
    this.removeEventListener('click', this.closeToolbelt);
  }
}

customElements.define("make-editable", MakeEditable);