import "@hotwired/turbo";
import { Application } from "@hotwired/stimulus";
import ChatInputController from "controllers/chat_input_controller.js";
import ChatModalController from "controllers/chat_modal_controller.js";

window.Stimulus = Application.start();
window.Stimulus.register("chat-input", ChatInputController);
window.Stimulus.register("chat-modal", ChatModalController);
