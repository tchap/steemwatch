import { Component, OnInit } from '@angular/core';

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
export class SteemitChatModalComponent implements OnInit {

  model = {username: '', password: ''};

  form: FormGroup;

  constructor(
    private formBuilder: FormBuilder,
    private chatService: SteemitChatService
  ) {}

  ngOnInit() {}
}
