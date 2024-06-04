import { LitElement, html, css, choose } from "/vendor/@lit/all@3.1.2/lit-all.min.js";

const iconDict = {
  /* Text */
  textColor: 'palette',
  bgColor: 'paint-bucket',
  editText: 'input-cursor-text',
  size: 'arrows-angle-expand',

  /* Image */
  borderColor: 'border-width',
  // bgColor: 'paint-bucket',
  crop: 'crop',
  effect: 'magic',
  imageReplace: 'upload',
}

const labelDict = {
  /* Text */
  textColor: 'Text',
  bgColor: 'Bg',
  editText: 'Edit',
  size: 'Sizing',

  /* Image */
  borderColor: 'Border',
  // bgColor: 'paint-bucket',
  crop: 'Crop',
  effect: 'Magic',
  imageReplace: 'Change',
}

function getToolIcon(o) {
  return iconDict[o] || 'question-square'
}

function getToolLabel(o) {
  return labelDict[o] || '??'
}

function getOptionHandler(o, ctx) {
  const handlerDict = {
    textColor: ctx.setTextColor,
    bgColor: ctx.setBgColor,
    editText: ctx.setText,
    size: ctx.setSize,

    borderColor: ctx.setBorderColor,
    crop: ctx.setCrop,
    effect: ctx.setMagic,
    imageReplace: ctx.setImage,
  }
  return handlerDict[o];
}

const primaryColors = ['white', 'red', 'orange', 'yellow', 'green', 'blue', 'violet', 'indigo', 'purple', 'black'];
let borderColorIndex = 0;
let colorIndex = 0;

export function setBorderColor(option, element) {
  element.style.borderColor = primaryColors[borderColorIndex];
  borderColorIndex = (borderColorIndex + 1) % primaryColors.length;
}

export function setTextColor(option, element) {
  element.style.color = primaryColors[colorIndex];
  colorIndex = (colorIndex + 1) % primaryColors.length;
}

export function generateOptions(element) {
  const name = element.getAttribute('data-edit-name');
  const type = element.getAttribute('data-edit-type');

  let options = [];
  
  switch(type) {
    case 'text':
      options = ['textColor', 'bgColor', 'editText', 'size'];
      break;

    case 'image':
      options = ['borderColor', 'bgColor', 'crop', 'effect', 'imageReplace'];
      break;
  }
  return options.map(option => html`
    ${choose(option, [
      [
        'borderColor', () => html`
          <option-color-picker 
            .forElement=${element}
            editName=${name}>
          </option-color-picker>`
      ],[
        'textColor', () => html`
        <div class="option" @click=${(event) => this.handleOptionClick(event, option, element)}>
          <sl-icon name="palette"></sl-icon>
          <span class="option-text">Text!</span>
        </div>`
      ]
    ],
    () => html`
      <div class="option" @click=${(event) => this.handleOptionClick(event, option, element)}>
        <sl-icon name="${getToolIcon(option)}"></sl-icon>
        <span class="option-text">${getToolLabel(option)}</span>
      </div>`)}
    `
  )
}

export function handleOptionClick(event, option, element) {
  event.stopPropagation();

  const handler = getOptionHandler(option, this)

  console.log(option, handler);
  if (typeof handler === 'function') {
    handler(option, element);
  } else {
    console.log('handler not found for option:', option);
  }
}