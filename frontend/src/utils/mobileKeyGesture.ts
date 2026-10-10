// Defer touch keys until release so a horizontal swipe never sends input.
export class MobileKeyGesture {
 private startX=0;private startY=0;private previousX=0;private moved=false;private active=false
 begin(x:number,y:number){this.startX=this.previousX=x;this.startY=y;this.moved=false;this.active=true}
 move(x:number,y:number):number{if(!this.active)return 0;const dx=x-this.previousX;this.previousX=x;if(Math.abs(x-this.startX)>8||Math.abs(y-this.startY)>8)this.moved=true;return this.moved&&Math.abs(x-this.startX)>=Math.abs(y-this.startY)?dx:0}
 finish():boolean{const tap=this.active&&!this.moved;this.active=false;return tap}
 cancel(){this.active=false}
}
