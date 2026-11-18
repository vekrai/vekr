import "@hotwired/turbo";
import { Application } from "@hotwired/stimulus";
import ChatModalController from "controllers/chat_modal_controller.js";

window.Stimulus = Application.start();
window.Stimulus.register("chat-modal", ChatModalController);
