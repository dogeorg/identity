import { html } from '/vendor/@lit/all@3.1.2/lit-all.min.js';

export function _render_profile_header(el, index) {
  return html`
  <make-editable container_id=${index}>
    <profile-header
      text="${el.vals.text}"
      text_color="${el.vals.text_color}"
      subtext="${el.vals.subtext}"
    ></profile-header>
  </make-editable>`
}