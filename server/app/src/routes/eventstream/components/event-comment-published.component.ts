import { Component, Input } from '@angular/core';


@Component({
  moduleId: module.id,
  selector: 'event-comment-published',
  templateUrl: 'event-comment-published.component.html'
})
export class CommentPublishedEventComponent {

  @Input() model: any;

  isRelated(account: string) : boolean {
    return (this.model.author === account);
  }
}
