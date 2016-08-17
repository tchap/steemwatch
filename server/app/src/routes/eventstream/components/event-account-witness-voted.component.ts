import { Component, Input } from '@angular/core';


@Component({
  moduleId: module.id,
  selector: 'event-account-witness-voted',
  templateUrl: 'event-account-witness-voted.component.html'
})
export class AccountWitnessVotedEventComponent {

  @Input() model: any;

  isRelated(account: string) : boolean {
    return (this.model.witness === account);
  }
}
