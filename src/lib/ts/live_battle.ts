export const TEAM_BLUE = 0;
export const TEAM_RED = 1;

export const SPEED = 150.0;
export const TURN_SPEED = 3.2;
export const TURN_COOLDOWN = 0.5;
export const VIEW_DISTANCE = 200.0;
export const FOV_RAD = (70.0 * Math.PI) / 180.0;
export const FIRE_INTERVAL = 0.55;
export const PROJECTILE_SPEED = 460.0;
export const DAMAGE = 1.0;
export const HULL_RADIUS = 9.0;
export const HULL_RADIUS_SQ = HULL_RADIUS * HULL_RADIUS;

export interface Projectile {
	x: number;
	y: number;
	rot: number;
	team: number;
	life: number;
}

export class Ship {
	x: number;
	y: number;
	rot: number;
	team: number;
	hp: number;
	fireCooldown: number;
	turnCooldown: number;
	pendingTurn: number;
	pendingDir: number;

	constructor(x: number, y: number, rot: number, team: number) {
		this.x = x;
		this.y = y;
		this.rot = rot;
		this.team = team;
		this.hp = 1.0;
		this.fireCooldown = 0;
		this.turnCooldown = 0;
		this.pendingTurn = 0;
		this.pendingDir = 0;
	}
}

export class BattleState {
	ships: Ship[] = [];
	projectiles: Projectile[] = [];
	width: number = 0;
	height: number = 0;
	spawnTimer: number = 0;
	
	init(w: number, h: number) {
		this.width = w;
		this.height = h;
		this.ships = [];
		this.projectiles = [];
		this.spawnTimer = 0;
		
		for(let i=0; i<8; i++) {
			this.spawnShip();
		}
	}

	spawnShip() {
		let blueCount = 0;
		let redCount = 0;
		for (const s of this.ships) {
			if (s.team === TEAM_BLUE) blueCount++;
			else if (s.team === TEAM_RED) redCount++;
		}

		if (blueCount >= 6 && redCount >= 6) return;

		let team = TEAM_BLUE;
		if (blueCount >= 6) {
			team = TEAM_RED;
		} else if (redCount >= 6) {
			team = TEAM_BLUE;
		} else {
			team = Math.random() > 0.5 ? TEAM_BLUE : TEAM_RED;
		}

		const edge = Math.floor(Math.random() * 4);
		let x = 0, y = 0, rot = 0;
		
		if (edge === 0) {
			x = Math.random() * this.width;
			y = -20;
			rot = Math.PI / 2;
		} else if (edge === 1) {
			x = this.width + 20;
			y = Math.random() * this.height;
			rot = Math.PI;
		} else if (edge === 2) {
			x = Math.random() * this.width;
			y = this.height + 20;
			rot = -Math.PI / 2;
		} else {
			x = -20;
			y = Math.random() * this.height;
			rot = 0;
		}
		
		this.ships.push(new Ship(x, y, rot, team));
	}

	update(delta: number) {
		this.spawnTimer -= delta;
		if (this.spawnTimer <= 0) {
			this.spawnShip();
			this.spawnTimer = 1.0 + Math.random() * 1.5;
		}

		for (let i = this.projectiles.length - 1; i >= 0; i--) {
			const p = this.projectiles[i];
			p.x += Math.cos(p.rot) * PROJECTILE_SPEED * delta;
			p.y += Math.sin(p.rot) * PROJECTILE_SPEED * delta;
			p.life -= delta;
			
			let hit = false;
			for (const s of this.ships) {
				if (s.team !== p.team) {
					const dx = s.x - p.x;
					const dy = s.y - p.y;
					if (dx * dx + dy * dy <= HULL_RADIUS_SQ) {
						s.hp = 0;
						hit = true;
						break;
					}
				}
			}

			if (hit || p.life <= 0 || p.x < -100 || p.x > this.width + 100 || p.y < -100 || p.y > this.height + 100) {
				this.projectiles.splice(i, 1);
			}
		}

		for (let i = this.ships.length - 1; i >= 0; i--) {
			if (this.ships[i].hp <= 0) {
				this.ships.splice(i, 1);
			}
		}

		for (const s of this.ships) {
			if (s.fireCooldown > 0) s.fireCooldown -= delta;
			if (s.turnCooldown > 0) s.turnCooldown -= delta;

			if (s.pendingTurn > 0) {
				const step = TURN_SPEED * delta;
				if (step >= s.pendingTurn) {
					s.rot += s.pendingDir * s.pendingTurn;
					s.pendingTurn = 0;
					s.turnCooldown = TURN_COOLDOWN;
				} else {
					s.rot += s.pendingDir * step;
					s.pendingTurn -= step;
				}
			} else {
				let nearestEnemy: Ship | null = null;
				let nearestEnemyDist = Infinity;
				let nearestAlly: Ship | null = null;
				let nearestAllyDist = Infinity;

				for (const other of this.ships) {
					if (other === s) continue;
					const dx = other.x - s.x;
					const dy = other.y - s.y;
					const distSq = dx * dx + dy * dy;
					
					if (distSq <= VIEW_DISTANCE * VIEW_DISTANCE) {
						let angleTo = Math.atan2(dy, dx);
						let diff = angleTo - s.rot;
						while (diff <= -Math.PI) diff += Math.PI * 2;
						while (diff > Math.PI) diff -= Math.PI * 2;
						
						if (Math.abs(diff) <= FOV_RAD * 0.5) {
							if (other.team !== s.team) {
								if (distSq < nearestEnemyDist) {
									nearestEnemyDist = distSq;
									nearestEnemy = other;
								}
							} else {
								if (distSq < nearestAllyDist) {
									nearestAllyDist = distSq;
									nearestAlly = other;
								}
							}
						}
					}
				}

				if (nearestEnemy) {
					const targetAngle = Math.atan2(nearestEnemy.y - s.y, nearestEnemy.x - s.x);
					this.turnToward(s, targetAngle, delta);
					
					if (s.fireCooldown <= 0) {
						this.projectiles.push({
							x: s.x + Math.cos(s.rot) * (HULL_RADIUS + 4),
							y: s.y + Math.sin(s.rot) * (HULL_RADIUS + 4),
							rot: s.rot,
							team: s.team,
							life: 3.0
						});
						s.fireCooldown = FIRE_INTERVAL;
					}
				} else if (nearestAlly) {
					const targetAngle = Math.atan2(nearestAlly.y - s.y, nearestAlly.x - s.x);
					this.turnToward(s, targetAngle, delta);
				} else {
					const dirX = Math.cos(s.rot);
					const dirY = Math.sin(s.rot);
					let distToRim = Infinity;
					
					if (dirX > 0.0001) distToRim = Math.min(distToRim, (this.width - s.x) / dirX);
					else if (dirX < -0.0001) distToRim = Math.min(distToRim, -s.x / dirX);
					
					if (dirY > 0.0001) distToRim = Math.min(distToRim, (this.height - s.y) / dirY);
					else if (dirY < -0.0001) distToRim = Math.min(distToRim, -s.y / dirY);

					const seesRim = distToRim <= VIEW_DISTANCE && (s.x > 0 && s.x < this.width && s.y > 0 && s.y < this.height);

					if (seesRim && s.turnCooldown <= 0) {
						s.pendingTurn = Math.PI;
						s.pendingDir = -1.0;
					}
				}
			}

			s.x += Math.cos(s.rot) * SPEED * delta;
			s.y += Math.sin(s.rot) * SPEED * delta;

			if (s.x < -400 || s.x > this.width + 400 || s.y < -400 || s.y > this.height + 400) {
				s.hp = 0;
			}
		}
	}

	turnToward(s: Ship, targetAngle: number, delta: number) {
		let diff = targetAngle - s.rot;
		while (diff <= -Math.PI) diff += Math.PI * 2;
		while (diff > Math.PI) diff -= Math.PI * 2;
		
		const step = TURN_SPEED * delta;
		if (Math.abs(diff) <= step) {
			s.rot = targetAngle;
		} else {
			s.rot += Math.sign(diff) * step;
		}
	}
}

export function drawBattle(ctx: CanvasRenderingContext2D, state: BattleState) {
	ctx.clearRect(0, 0, state.width, state.height);

	for (const p of state.projectiles) {
		ctx.beginPath();
		ctx.arc(p.x, p.y, 2.5, 0, Math.PI * 2);
		ctx.fillStyle = p.team === TEAM_BLUE ? '#739dff' : '#ff6b52';
		ctx.fill();
	}

	for (const s of state.ships) {
		ctx.save();
		ctx.translate(s.x, s.y);
		ctx.rotate(s.rot);

		ctx.beginPath();
		ctx.moveTo(0, 0);
		ctx.arc(0, 0, VIEW_DISTANCE, -FOV_RAD / 2, FOV_RAD / 2);
		ctx.lineTo(0, 0);
		ctx.fillStyle = s.team === TEAM_BLUE ? 'rgba(115, 157, 255, 0.05)' : 'rgba(255, 107, 82, 0.05)';
		ctx.fill();

		ctx.beginPath();
		ctx.moveTo(13, 0);
		ctx.lineTo(-9, -7);
		ctx.lineTo(-9, 7);
		ctx.closePath();
		
		ctx.fillStyle = s.team === TEAM_BLUE ? '#739dff' : '#ff6b52';
		ctx.fill();
		ctx.lineWidth = 1.5;
		ctx.strokeStyle = '#0d0d1a';
		ctx.stroke();

		ctx.restore();
	}
}
