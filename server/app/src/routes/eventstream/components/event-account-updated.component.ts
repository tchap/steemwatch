import { Component, Input } from '@angular/core';


@Component({
  moduleId: module.id,
  selector: 'event-account-updated',
  templateUrl: 'event-account-updated.component.html'
})
export class AccountUpdatedEventComponent {

  @Input() model: any;
}
