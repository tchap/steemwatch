import { Component, Input, ViewChild, OnInit } from '@angular/core';

import { MessageService } from '../../../services/index';

import { TelegramService } from '../services/telegram.service';
import { TelegramModel }   from '../models/telegram.model';


@Component({
  moduleId: module.id,
  selector: 'telegram',
  templateUrl: 'telegram.component.html',
  styleUrls: ['telegram.component.css'],
  providers: [TelegramService]
})
export class TelegramComponent implements OnInit {

  model: TelegramModel;

  processing: boolean;
  errorMessage: string;

  constructor(
    private telegramService: TelegramService,
    private messageService: MessageService
  ) {}

  ngOnInit() {
    this.telegramService.load()
      .subscribe(
        model => {
          this.model = model;
        },
        err => this.messageService.error(err)
      );
  }

  setEnabled(enabled: boolean) : void {
    this.processing = true;
    this.errorMessage = null;

    this.telegramService.setEnabled(enabled)
      .finally(() => this.processing = false)
      .subscribe(
        () => this.model.enabled = enabled,
        (err) => this.errorMessage = `${err.message || err}`
      );
  }

  disconnect() : void {
    this.processing = true;
    this.errorMessage = null;

    this.telegramService.disconnect()
      .finally(() => this.processing = false)
      .subscribe(
        () => this.resetModel(),
        (err) => this.errorMessage = `${err.message || err}`
      );
  }

  private resetModel() : void {
    this.model = {
      enabled: false,
      settings: {
        startToken: this.model.settings.startToken,
        firstName: null,
        lastName: null,
        username: null,
      }
    };
  }
}
