import { Component, Input, ViewChild, OnInit } from '@angular/core';

import { MessageService } from '../../../services/index';

import { SlackService } from '../services/slack.service';
import { SlackModel }   from '../models/slack.model';


@Component({
  moduleId: module.id,
  selector: 'slack',
  templateUrl: 'slack.component.html',
  providers: [SlackService]
})
export class SlackComponent implements OnInit {

  model: SlackModel;
  storedWebhookURL: string;
  notSlackURL: boolean;
  dirty: boolean;

  @ViewChild('input') input;
  formActive: boolean = true;

  saving: boolean = false;
  errorMessage: string;

  constructor(
    private slackService: SlackService,
    private messageService: MessageService
  ) {}

  ngOnInit() {
    this.slackService.load()
      .subscribe(
        model => {
          this.model = model;
          this.storedWebhookURL = model.settings.webhookURL;
        },
        err => this.messageService.error(err)
      );
  }

  inputChanged() {
    const webhookURL = this.model.settings.webhookURL;

    this.notSlackURL = !webhookURL.startsWith('https://hooks.slack.com/services');

    this.dirty = webhookURL !== this.storedWebhookURL;

    if (this.storedWebhookURL && !this.dirty) {
      this.formActive = false;
      setTimeout(() => {
        this.formActive = true;
        setTimeout(() => {
          this.input.nativeElement.focus();
        }, 0);
      }, 0);
    }
  }

  onSubmit() {
    this.saving = true;
    const enabled = this.model.enabled;
    this.model.enabled = true;
    this.model.settings.webhookURL = this.model.settings.webhookURL.trim();

    this.slackService.save(this.model)
      .subscribe(
        () => {
          this.storedWebhookURL = this.model.settings.webhookURL;
          this.dirty = false;
          this.saving = false;
          this.errorMessage = null;
          setTimeout(() => this.inputChanged(), 0);
        },
        (err) => {
          this.model.enabled = enabled;
          this.saving = false;
          this.errorMessage = `${err.status} ${err.text()}`;
        }
      );
  }

  toggleEnabled() {
    this.saving = true;

    this.slackService.update({enabled: !this.model.enabled})
      .subscribe(
        () => {
          this.model.enabled = !this.model.enabled;
          this.saving = false;
          this.errorMessage = null;
        },
        (err) => {
          this.saving = false;
          this.errorMessage = `${err.status} ${err.text()}`
        }
      );
  }
}
