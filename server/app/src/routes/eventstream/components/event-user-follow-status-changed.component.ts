import { Component, Input } from '@angular/core';


@Component({
  moduleId: module.id,
  selector: 'event-user-follow-status-changed',
  templateUrl: 'event-user-follow-status-changed.component.html'
})
export class UserFollowStatusChangedEventComponent {

  @Input() model: any;

  isRelated(account: string) : boolean {
    return (this.model.following === account);
  }
}
