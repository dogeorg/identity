import { html, css } from "/vendor/@lit/all@3.1.2/lit-all.min.js";

export function renderNav() {
  const topNavStyles = css`
    nav.top {
      position: absolute;
      top: 0px;
      left: 0px;
      z-index: 100;

      padding: 1em;

      font-family: "Comic Neue";

      a {
        color: white;
      }
    }
  `;

  return html`
    <nav class="top">
      <!-- a href="/">← Back to Editing</a -->
    </nav>
    <style>
      ${topNavStyles}
    </style>
  `;
}
