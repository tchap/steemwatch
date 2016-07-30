import { Component, ViewChild } from '@angular/core';

import {
  REACTIVE_FORM_DIRECTIVES,
  FormGroup,
  FormControl,
  FormBuilder
} from '@angular/forms';

import { SteemitChatService } from '../services/steemit-chat.service';


@Component({
  moduleId: module.id,
  selector: 'steemit-chat-modal',
  templateUrl: 'steemit-chat-modal.component.html',
  styleUrls: ['steemit-chat-modal.component.css'],
  directives: [REACTIVE_FORM_DIRECTIVES],
  providers: [SteemitChatService]
})
export class SteemitChatModalComponent {

  @ViewChild('closeButton') closeButton;

  model = {username: '', password: ''};
  saving: boolean;

  form: FormGroup;

  constructor(
    private formBuilder: FormBuilder,
    private chatService: SteemitChatService
  ) {}

  onSubmit() {
    this.closeModal();
    this.resetModel();
  }

  private closeModal() : void {
    setTimeout(() => {
      const evt = new MouseEvent('click', {bubbles: true});
      this.closeButton.nativeElement.dispatchEvent(evt);
    }, 0);
  }

  private resetModel() : void {
    this.model = {username: '', password: ''};
  }
}
