import { Component, Input } from '@angular/core';


@Component({
  moduleId: module.id,
  selector: 'event-user-mentioned',
  templateUrl: 'event-user-mentioned.component.html'
})
export class UserMentionedEventComponent {

  @Input() model: any;

  isRelated(account: string) : boolean {
    return (this.model.user === account);
  }
}
