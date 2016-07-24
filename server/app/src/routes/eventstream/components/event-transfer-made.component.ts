import { Component, Input } from '@angular/core';


@Component({
  moduleId: module.id,
  selector: 'event-transfer-made',
  templateUrl: 'event-transfer-made.component.html',
  styleUrls: ['event-transfer-made.component.css']
})
export class TransferMadeEventComponent {

  @Input() model: any;
}
