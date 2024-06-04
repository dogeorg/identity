import { css } from "/vendor/@lit/all@3.1.2/lit-all.min.js";

export const defaultOptionStyles = css`
  :host {
    position: relative;
    display: flex;
    flex-direction: column;
    gap: 4px;
    align-items: center;
    justify-content: center;
    width: 40px;
    height: 55px;
    padding: 8px;
    padding-bottom: 4px;
    color: black;
    user-select: none;
  }

  sl-icon {
    font-size: 1.35rem;
  }

  .option-text {
    font-size: 0.7rem;
    text-transform: uppercase;
    font-weight: bold;
    user-select: none;
  }

  :host(:hover) {
    cursor: pointer;
    background: rgba(255,255,255, 0.25);
  }
`