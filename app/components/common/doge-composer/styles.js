import { css } from "/vendor/@lit/all@3.1.2/lit-all.min.js";

export const styles = css`
  :host {
    display: block;
    position: relative;
  }

  .elements-container {
    display: block;
    padding: 20px;
    border: 1px dashed #ccc;
    max-width: 480px;
  }

  .element-container {
    position: relative;
    display: block;
    padding: 2px;
    transition-property: border, outline;
    transition-duration: 100ms;
    transition-timing-function: ease-out;
    scale: 1;
    border: 1px solid #444;
  }
  .element-container:hover {
    border-color: transparent;
    outline: 2px solid yellow;
    outline-offset: 0px;
  }
  .element-container.actively-editing {
    z-index: 99;
  }

  /* DEBUG PANEL */
  .floating-aside {
    position: absolute;
    left: 600px;
    top: 50px;
    display: flex;
    flex-direction: row;
    gap: 5em;
  }

  .floating-aside > div {
    overflow-x: auto;
    max-width: 380px;
    min-width: 200px;
  }

`
