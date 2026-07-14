// tetris3d.js - 3D Тетрис на JavaScript (Node.js)
const fs = require('fs');
const keypress = require('keypress');
const readline = require('readline');

const WIDTH = 6;
const DEPTH = 6;
const HEIGHT = 6;
const EMPTY = 0;
const RECORD_FILE = 'tetris3d_record.json';

// Фигуры
const SHAPES = {
    I: [[0,0,0],[1,0,0],[2,0,0],[3,0,0]],
    O: [[0,0,0],[1,0,0],[0,1,0],[1,1,0]],
    T: [[0,0,0],[1,0,0],[2,0,0],[1,1,0]],
    L: [[0,0,0],[1,0,0],[2,0,0],[2,1,0]],
    J: [[0,0,0],[1,0,0],[2,0,0],[0,1,0]],
    S: [[1,0,0],[2,0,0],[0,1,0],[1,1,0]],
    Z: [[0,0,0],[1,0,0],[1,1,0],[2,1,0]],
};

class Tetris3D {
    constructor() {
        this.field = Array.from({ length: HEIGHT }, () =>
            Array.from({ length: WIDTH }, () => Array(DEPTH).fill(EMPTY))
        );
        this.score = 0;
        this.speed = 1;
        this.gameOver = false;
        this.paused = false;
        this.running = true;
        this.record = this.loadRecord();
        this.currentShape = null;
        this.shapePos = [0,0,0];
        this.dropTimer = Date.now();
        this.lock = false;
    }

    loadRecord() {
        try {
            const data = fs.readFileSync(RECORD_FILE, 'utf8');
            return JSON.parse(data).record || 0;
        } catch { return 0; }
    }

    saveRecord() {
        fs.writeFileSync(RECORD_FILE, JSON.stringify({ record: this.record }));
    }

    randomShape() {
        const names = Object.keys(SHAPES);
        const name = names[Math.floor(Math.random() * names.length)];
        return SHAPES[name].map(p => [...p]);
    }

    spawnShape() {
        const shape = this.randomShape();
        const xs = shape.map(p => p[0]);
        const zs = shape.map(p => p[1]);
        const cx = Math.floor((Math.max(...xs) + Math.min(...xs)) / 2);
        const cz = Math.floor((Math.max(...zs) + Math.min(...zs)) / 2);
        const startX = Math.floor(WIDTH / 2) - cx;
        const startZ = Math.floor(DEPTH / 2) - cz;
        const startY = HEIGHT - 1;
        // Проверка
        for (const [dx, dz, dy] of shape) {
            const x = startX + dx, z = startZ + dz, y = startY + dy;
            if (x < 0 || x >= WIDTH || z < 0 || z >= DEPTH || y < 0 || y >= HEIGHT || this.field[y][x][z] !== EMPTY) {
                this.gameOver = true;
                if (this.score > this.record) {
                    this.record = this.score;
                    this.saveRecord();
                }
                return false;
            }
        }
        this.currentShape = shape;
        this.shapePos = [startX, startZ, startY];
        return true;
    }

    collides(shape, pos) {
        for (const [dx, dz, dy] of shape) {
            const x = pos[0] + dx, z = pos[1] + dz, y = pos[2] + dy;
            if (x < 0 || x >= WIDTH || z < 0 || z >= DEPTH || y < 0 || y >= HEIGHT) return true;
            if (this.field[y][x][z] !== EMPTY) return true;
        }
        return false;
    }

    lockShape() {
        for (const [dx, dz, dy] of this.currentShape) {
            const x = this.shapePos[0] + dx, z = this.shapePos[1] + dz, y = this.shapePos[2] + dy;
            this.field[y][x][z] = 1;
        }
        // Удаление слоёв
        let cleared = 0;
        for (let y = 0; y < HEIGHT; y++) {
            if (this.field[y].every(row => row.every(cell => cell !== EMPTY))) {
                for (let y2 = y; y2 > 0; y2--) {
                    this.field[y2] = this.field[y2-1].map(row => [...row]);
                }
                this.field[0] = Array.from({ length: WIDTH }, () => Array(DEPTH).fill(EMPTY));
                cleared++;
            }
        }
        if (cleared) {
            this.score += cleared;
            this.speed = 1 + Math.floor(this.score / 3);
        }
        if (!this.spawnShape()) this.gameOver = true;
    }

    move(dx, dz, dy) {
        const newPos = [this.shapePos[0] + dx, this.shapePos[1] + dz, this.shapePos[2] + dy];
        if (!this.collides(this.currentShape, newPos)) {
            this.shapePos = newPos;
            return true;
        }
        return false;
    }

    rotate() {
        const newShape = this.currentShape.map(([x, z, y]) => [-z, x, y]);
        if (!this.collides(newShape, this.shapePos)) {
            this.currentShape = newShape;
            return true;
        }
        return false;
    }

    drop() {
        while (this.move(0, 0, -1)) {}
        this.lockShape();
    }

    update() {
        if (this.gameOver || this.paused) return;
        if (Date.now() - this.dropTimer >= 1000 / this.speed) {
            if (!this.move(0, 0, -1)) {
                this.lockShape();
            }
            this.dropTimer = Date.now();
        }
    }

    getFieldWithShape() {
        const fieldCopy = this.field.map(plane => plane.map(row => [...row]));
        if (this.currentShape) {
            for (const [dx, dz, dy] of this.currentShape) {
                const x = this.shapePos[0] + dx, z = this.shapePos[1] + dz, y = this.shapePos[2] + dy;
                if (x >= 0 && x < WIDTH && z >= 0 && z < DEPTH && y >= 0 && y < HEIGHT) {
                    fieldCopy[y][x][z] = 2;
                }
            }
        }
        return fieldCopy;
    }

    draw() {
        console.clear();
        const field = this.getFieldWithShape();
        console.log('═'.repeat(WIDTH * 2 + 10));
        console.log(`  Счёт: ${this.score}   Рекорд: ${this.record}   Скорость: ${this.speed}`);
        console.log('═'.repeat(WIDTH * 2 + 10));
        for (let y = HEIGHT - 1; y >= 0; y--) {
            process.stdout.write(`  Y=${y}  `);
            for (let z = 0; z < DEPTH; z++) {
                for (let x = 0; x < WIDTH; x++) {
                    const val = field[y][x][z];
                    if (val === 2) process.stdout.write('\x1b[32m█ \x1b[0m');
                    else if (val === 1) process.stdout.write('\x1b[36m█ \x1b[0m');
                    else process.stdout.write('· ');
                }
                process.stdout.write('  ');
            }
            console.log();
        }
        console.log('═'.repeat(WIDTH * 2 + 10));
        const status = this.paused ? 'ПАУЗА' : this.gameOver ? 'ИГРА ОКОНЧЕНА' : 'ИГРА';
        console.log(`  ${status}  |  ←/→ - X  |  W/S - Z  |  Пробел - вращение  |  ↓ - падение`);
        console.log(`  P - пауза  |  R - рестарт  |  Q - выход`);
    }

    reset() {
        this.field = Array.from({ length: HEIGHT }, () =>
            Array.from({ length: WIDTH }, () => Array(DEPTH).fill(EMPTY))
        );
        this.score = 0;
        this.speed = 1;
        this.gameOver = false;
        this.paused = false;
        this.spawnShape();
    }

    handleKey(ch, key) {
        if (!key) return;
        if (this.lock) return;
        this.lock = true;
        const action = () => {
            if (key.name === 'left') { if (!this.gameOver && !this.paused) this.move(-1,0,0); }
            else if (key.name === 'right') { if (!this.gameOver && !this.paused) this.move(1,0,0); }
            else if (key.name === 'w') { if (!this.gameOver && !this.paused) this.move(0,-1,0); }
            else if (key.name === 's') { if (!this.gameOver && !this.paused) this.move(0,1,0); }
            else if (key.name === 'space') { if (!this.gameOver && !this.paused) this.rotate(); }
            else if (key.name === 'down') { if (!this.gameOver && !this.paused) this.drop(); }
            else if (key.name === 'p') { this.paused = !this.paused; }
            else if (key.name === 'r') { this.reset(); }
            else if (key.name === 'q') { this.running = false; process.stdin.pause(); process.exit(0); }
            this.lock = false;
        };
        setTimeout(action, 10);
    }

    run() {
        this.spawnShape();
        keypress(process.stdin);
        process.stdin.setRawMode(true);
        process.stdin.resume();
        process.stdin.on('keypress', (ch, key) => this.handleKey(ch, key));

        const gameLoop = () => {
            if (!this.running) return;
            this.update();
            this.draw();
            setTimeout(gameLoop, 1000 / 30);
        };
        gameLoop();
    }
}

const game = new Tetris3D();
game.run();
