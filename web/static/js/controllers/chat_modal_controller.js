import { Controller } from "@hotwired/stimulus";

export default class extends Controller {
  static targets = ["modal", "messages", "input"];
  static values = { chatId: Number };

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
    this.sync();
    if (this.hasInputTarget && !this.inputTarget.disabled) this.inputTarget.focus();
  }

  openNew(event) {
    if (!this.chatIdValue) {
      this.open(event);
      return;
    }
    if (this.visiting) return;
    this.visiting = true;
    event?.target?.blur?.();
    window.Turbo.visit(this.urlWith({ chat: "new" }), { action: "replace" });
  }

  minimize() {
    this.modalTarget.hidden = true;
    this.sync();
  }

  backdrop(event) {
    if (event.target === this.modalTarget) this.minimize();
  }

  sync() {
    document.body.classList.toggle("overflow-hidden", this.isOpen);
    const wanted = this.isOpen && this.chatIdValue ? this.urlWith({ chat_id: this.chatIdValue }) : this.urlWith({});
    if (wanted !== location.pathname + location.search) history.replaceState(history.state, "", wanted);
    if (this.isOpen) {
      this.scrollToBottom();
      if (this.hasInputTarget && !this.inputTarget.disabled && document.activeElement === document.body) this.inputTarget.focus();
    }
  }

  urlWith(params) {
    const url = new URL(location.href);
    url.searchParams.delete("chat_id");
    url.searchParams.delete("chat");
    for (const [key, value] of Object.entries(params)) url.searchParams.set(key, value);
    return url.pathname + url.search;
  }

  scrollToBottom() {
    if (this.hasMessagesTarget) this.messagesTarget.scrollTop = this.messagesTarget.scrollHeight;
  }
}
