import { Controller } from "@hotwired/stimulus";

export default class extends Controller {
  static targets = ["modal", "messages", "input"];
  static values = { url: String };

  connect() {
    this.visiting = false;
    this.onKeydown = (event) => {
      if (event.key === "Escape" && this.isOpen) this.minimize();
    };
    document.addEventListener("keydown", this.onKeydown);
    this.sync();
  }

  disconnect() {
    document.removeEventListener("keydown", this.onKeydown);
    document.body.classList.remove("overflow-hidden");
  }

  messagesTargetConnected() {
    this.scrollToBottom();
  }

  get isOpen() {
    return !this.modalTarget.hidden;
  }

  open(event) {
    event?.target?.blur?.();
    this.modalTarget.hidden = false;
    history.replaceState(history.state, "", this.urlValue);
    this.sync();
    if (this.hasInputTarget && !this.inputTarget.disabled) this.inputTarget.focus();
  }

  openNew(event) {
    if (this.urlValue === "/") {
      this.open(event);
      return;
    }
    if (this.visiting) return;
    this.visiting = true;
    event?.target?.blur?.();
    window.Turbo.visit("/?chat=new", { action: "replace" });
  }

  minimize() {
    this.modalTarget.hidden = true;
    history.replaceState(history.state, "", "/");
    this.sync();
  }

  backdrop(event) {
    if (event.target === this.modalTarget) this.minimize();
  }

  sync() {
    document.body.classList.toggle("overflow-hidden", this.isOpen);
    if (this.isOpen) {
      if (location.pathname + location.search !== this.urlValue) history.replaceState(history.state, "", this.urlValue);
      this.scrollToBottom();
      if (this.hasInputTarget && !this.inputTarget.disabled && document.activeElement === document.body) this.inputTarget.focus();
    }
  }

  scrollToBottom() {
    if (this.hasMessagesTarget) this.messagesTarget.scrollTop = this.messagesTarget.scrollHeight;
  }
}
