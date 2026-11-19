import { Controller } from "@hotwired/stimulus";

export default class extends Controller {
  connect() {
    this.submitting = false;
  }

  keydown(event) {
    if (event.key !== "Enter" || event.isComposing || event.keyCode === 229) return;

    if (event.altKey) {
      event.preventDefault();
      this.insertNewline();
      return;
    }
    if (event.shiftKey) return;

    event.preventDefault();
    if (this.submitting || this.element.disabled || this.element.value.trim() === "") return;
    this.submitting = true;
    this.element.form.requestSubmit();
  }

  insertNewline() {
    const { selectionStart, selectionEnd, value } = this.element;
    this.element.value = value.slice(0, selectionStart) + "\n" + value.slice(selectionEnd);
    this.element.selectionStart = this.element.selectionEnd = selectionStart + 1;
    this.element.dispatchEvent(new Event("input", { bubbles: true }));
  }
}
