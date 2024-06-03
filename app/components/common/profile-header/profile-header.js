import { LitElement, html, css, styleMap } from '/vendor/@lit/all@3.1.2/lit-all.min.js';

class ProfileHeader extends LitElement {
  static properties = {
    // Background
    bg_color: { type: String },
    bg_border_color: { type: String },
    bg_img_url: { type: String },
    bg_size: { type: String },
    bg_anchor: { type: String },
    bg_filter: { type: String },
    bg_opacity: { type: Number },

    // Avatar
    avatar_border_color: { type: String },
    avatar_bg_color: { type: String },
    avatar_img_url: { type: String },
    avatar_bg_size: { type: String },

    // Text
    text: { type: String },
    text_color: { type: String },
  }

  static styles = css`
    :host {
      position: relative;
      overflow: hidden;
      display: block;
      height:270px;
      width: 100%;
    }
    .background-wrap {
      position:absolute;
      z-index: -1;
      top: 0;
      left: 0;
      width: 100%;
      height: 100%;
    }
    .background-wrap .background {
      display: block;
      width: 100%;
      height: 100%;
      background-color: #aaa;
      background-image: url("/static/img/profile-header/bg.png");
      background-size: contain;
      opacity: 0.4;
    }

    .subject-wrap {
      display: flex;
      width: 100%;
      height: 100%;
      flex-direction: column;
      align-items: center;
      justify-content: center;
    }

    .avatar {
      height: 120px;
      width: 120px;
      border: 3px solid black;
      background-color: #ddd;
      background-image: url("/static/img/profile-header/avatar.png");
      background-size: contain;
      transition: transform 100ms ease-out;
    }

    .avatar:hover {
      transform: rotate(3deg);
    }

    .text {
      font-family: 'Comic Neue';
      font-weight: bold;
      color: white;
      font-size: 2rem;

      text-shadow: 1px 1px 2px rgba(0,0,0,0.2);
    }
  `

  constructor() {
    super();
    this.text_default = "Mystery Shibe"
  }

  render() {
    const bg_styles = {
      backgroundImage: this.bg_img_url,
      backgroundSize: this.bg_size,
      opacity: this.bg_opacity,
    }

    const avatar_styles = {
      borderColor: this.avatar_border_color,
      backgroundColor: this.avatar_bg_color,
      backgroundImage: this.avatar_img_url,
      backgroundSize: this.avatar_bg_size,
    }

    const text_styles = {
      color: this.text_color
    }

    return html`
      <div class="background-wrap">
        <div
          class="background"
          style=${styleMap(bg_styles)}
          data-edit-type="background-image"
          data-edit-name="bg"
        ></div>
      </div>
      <div class="subject-wrap">
        <div class="avatar-wrap">
          <div
            class="avatar"
            style=${styleMap(avatar_styles)}
            data-edit-type="image"
            data-edit-name="avatar"
          ></div>
        </div>
        <div class="text-wrap">
          <div
            class="text"
            data-edit-type="text"
            data-edit-name="text"
            style=${styleMap(text_styles)}
          >${this.text || this.text_default}</div>
        </div>
      </div>
      `
  }
}

customElements.define('profile-header', ProfileHeader);


