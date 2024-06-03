import { html } from '/vendor/@lit/all@3.1.2/lit-all.min.js';

export function _render_profile_header(el, index) {
  return html`
  <make-editable container_id=${index}>
    <profile-header></profile-header>
  </make-editable>`
}