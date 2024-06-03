import {
  LitElement,
  html,
  css
} from "/vendor/@lit/all@3.1.2/lit-all.min.js";

class EditableToolbelt extends LitElement {
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
      --arrow-color: #dd19c8;
      --arrow-size: 0.7rem;
    }

    sl-popup::part(popup) {
      box-shadow: 2px 2px 4px rgba(0,0,0,0.1);
    }

    .box {
      display: flex;
      justify-content: center;
      align-items: start;
      background: #dd19c8;
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
      padding: 8px;
      padding-bottom: 4px;
      color: black;
    }

    .option sl-icon {
      font-size: 1.35rem;
    }

    .option .option-text {
      font-size: 0.7rem;
      text-transform: uppercase;
      font-weight: bold;
    }

    .option:hover {
      cursor: pointer;
      background: rgba(0,0,0,0.2);
    }
  `;

  constructor() {
    super();
    this.forElement = null;
    this.editName = '';
  }

  connectedCallback() {
    super.connectedCallback();
  }

  firstUpdated() {
    this.attachOptionListeners();
  }

  show() {
      this.active = true;
    }

  hide() {
    this.active = false;
  }

  attachOptionListeners() {
    const optionsButtons = this.shadowRoot.querySelectorAll('.box .options .option');
    optionsButtons.forEach((element, index) => {
      element.addEventListener('click', (event) => this.handleOptionsClick(event));
    });
  }

  handleOptionsClick(event) {
    console.log(event);
    event.stopPropagation();
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
            <div class="option">
              <sl-icon name="arrows-move"></sl-icon>
              <span class="option-text">Move</span>
            </div>
            <div class="option">
              <sl-icon name="brush"></sl-icon>
              <span class="option-text">Edit</span>
            </div>
            <div class="option">
              <sl-icon name="trash3"></sl-icon>
              <span class="option-text">Trash</span>
            </div>
            <div class="option">
              <sl-icon name="copy"></sl-icon>
              <span class="option-text">Copy</span>
            </div>
          </div>
          <div class="drawer"></div>
        </div>
      </sl-popup>
    `;
  }
}

customElements.define("editable-toolbelt", EditableToolbelt);