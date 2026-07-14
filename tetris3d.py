
---

## 💻 Код на 7 языках

### 1. Python – `tetris3d.py`

```python
#!/usr/bin/env python3
# tetris3d.py - 3D Тетрис на Python

import os
import sys
import time
import random
import json
import threading
from copy import deepcopy
from colorama import init, Fore, Style

init(autoreset=True)

# Константы
WIDTH = 6      # X
DEPTH = 6      # Z
HEIGHT = 6     # Y
EMPTY = 0
FIELD = [[[EMPTY for _ in range(DEPTH)] for _ in range(WIDTH)] for _ in range(HEIGHT)]
RECORD_FILE = 'tetris3d_record.json'

# Фигуры: список координат (x, z, y) относительных (локальных)
SHAPES = {
    'I': [(0,0,0), (1,0,0), (2,0,0), (3,0,0)],
    'O': [(0,0,0), (1,0,0), (0,1,0), (1,1,0)],
    'T': [(0,0,0), (1,0,0), (2,0,0), (1,1,0)],
    'L': [(0,0,0), (1,0,0), (2,0,0), (2,1,0)],
    'J': [(0,0,0), (1,0,0), (2,0,0), (0,1,0)],
    'S': [(1,0,0), (2,0,0), (0,1,0), (1,1,0)],
    'Z': [(0,0,0), (1,0,0), (1,1,0), (2,1,0)],
}

class Tetris3D:
    def __init__(self):
        self.field = deepcopy(FIELD)
        self.score = 0
        self.speed = 1
        self.game_over = False
        self.paused = False
        self.running = True
        self.record = self.load_record()
        self.current_shape = None
        self.shape_pos = [0, 0, 0]  # x, z, y
        self.drop_timer = time.time()
        self.lock = threading.Lock()

    def load_record(self):
        try:
            with open(RECORD_FILE, 'r') as f:
                return json.load(f).get('record', 0)
        except:
            return 0

    def save_record(self):
        with open(RECORD_FILE, 'w') as f:
            json.dump({'record': self.record}, f)

    def random_shape(self):
        name = random.choice(list(SHAPES.keys()))
        shape = [(x, z, y) for x, z, y in SHAPES[name]]
        return shape

    def spawn_shape(self):
        shape = self.random_shape()
        # Центрирование по X и Z
        xs = [p[0] for p in shape]
        zs = [p[1] for p in shape]
        cx = (max(xs) + min(xs)) // 2
        cz = (max(zs) + min(zs)) // 2
        start_x = WIDTH // 2 - cx
        start_z = DEPTH // 2 - cz
        start_y = HEIGHT - 1  # сверху
        # Проверка возможности спавна
        for dx, dz, dy in shape:
            x = start_x + dx
            z = start_z + dz
            y = start_y + dy
            if not (0 <= x < WIDTH and 0 <= z < DEPTH and 0 <= y < HEIGHT) or self.field[y][x][z] != EMPTY:
                self.game_over = True
                if self.score > self.record:
                    self.record = self.score
                    self.save_record()
                return False
        self.current_shape = shape
        self.shape_pos = [start_x, start_z, start_y]
        return True

    def collides(self, shape, pos):
        for dx, dz, dy in shape:
            x = pos[0] + dx
            z = pos[1] + dz
            y = pos[2] + dy
            if not (0 <= x < WIDTH and 0 <= z < DEPTH and 0 <= y < HEIGHT):
                return True
            if self.field[y][x][z] != EMPTY:
                return True
        return False

    def lock_shape(self):
        for dx, dz, dy in self.current_shape:
            x = self.shape_pos[0] + dx
            z = self.shape_pos[1] + dz
            y = self.shape_pos[2] + dy
            self.field[y][x][z] = 1  # блок
        # Проверка заполненных слоёв
        cleared = 0
        for y in range(HEIGHT):
            if all(self.field[y][x][z] != EMPTY for x in range(WIDTH) for z in range(DEPTH)):
                # Удалить слой
                for y2 in range(y, 0, -1):
                    self.field[y2] = self.field[y2-1]
                self.field[0] = [[EMPTY for _ in range(DEPTH)] for _ in range(WIDTH)]
                cleared += 1
        if cleared > 0:
            self.score += cleared
            self.speed = 1 + self.score // 3
        # Спавн новой фигуры
        if not self.spawn_shape():
            self.game_over = True

    def move(self, dx, dz, dy):
        new_pos = [self.shape_pos[0] + dx, self.shape_pos[1] + dz, self.shape_pos[2] + dy]
        if not self.collides(self.current_shape, new_pos):
            self.shape_pos = new_pos
            return True
        return False

    def rotate(self):
        # Вращение вокруг оси Y (меняем x и z)
        new_shape = []
        for x, z, y in self.current_shape:
            new_shape.append((-z, x, y))  # поворот на 90° по часовой
        if not self.collides(new_shape, self.shape_pos):
            self.current_shape = new_shape
            return True
        return False

    def drop(self):
        while self.move(0, 0, -1):
            pass
        self.lock_shape()

    def update(self):
        if self.game_over or self.paused:
            return
        # Автопадание
        if time.time() - self.drop_timer >= 1.0 / self.speed:
            if not self.move(0, 0, -1):
                self.lock_shape()
            self.drop_timer = time.time()

    def get_field_with_shape(self):
        field_copy = deepcopy(self.field)
        if self.current_shape:
            for dx, dz, dy in self.current_shape:
                x = self.shape_pos[0] + dx
                z = self.shape_pos[1] + dz
                y = self.shape_pos[2] + dy
                if 0 <= x < WIDTH and 0 <= z < DEPTH and 0 <= y < HEIGHT:
                    field_copy[y][x][z] = 2  # фигура
        return field_copy

    def draw(self):
        os.system('cls' if os.name == 'nt' else 'clear')
        print('═' * (WIDTH * 2 + 10))
        print(f'  Счёт: {self.score}   Рекорд: {self.record}   Скорость: {self.speed}')
        print('═' * (WIDTH * 2 + 10))
        # Отображение слоёв (сверху вниз)
        for y in range(HEIGHT-1, -1, -1):
            print(f'  Y={y}  ', end='')
            for z in range(DEPTH):
                for x in range(WIDTH):
                    val = self.field[y][x][z]
                    if val == 2:
                        print(Fore.GREEN + '█' + Style.RESET_ALL, end=' ')
                    elif val == 1:
                        print(Fore.CYAN + '█' + Style.RESET_ALL, end=' ')
                    else:
                        print('·', end=' ')
                print('  ', end='')
            print()
        # Отображение текущей фигуры (проекция)
        print('═' * (WIDTH * 2 + 10))
        status = "ПАУЗА" if self.paused else ("ИГРА ОКОНЧЕНА" if self.game_over else "ИГРА")
        print(f'  {status}  |  ←/→ - X  |  W/S - Z  |  Пробел - вращение  |  ↓ - падение')
        print(f'  P - пауза  |  R - рестарт  |  Q - выход')

    def reset(self):
        self.field = deepcopy(FIELD)
        self.score = 0
        self.speed = 1
        self.game_over = False
        self.paused = False
        self.spawn_shape()

    def handle_input(self):
        while self.running:
            try:
                if keyboard.is_pressed('left'):
                    with self.lock:
                        if not self.game_over and not self.paused:
                            self.move(-1, 0, 0)
                elif keyboard.is_pressed('right'):
                    with self.lock:
                        if not self.game_over and not self.paused:
                            self.move(1, 0, 0)
                elif keyboard.is_pressed('w'):
                    with self.lock:
                        if not self.game_over and not self.paused:
                            self.move(0, -1, 0)
                elif keyboard.is_pressed('s'):
                    with self.lock:
                        if not self.game_over and not self.paused:
                            self.move(0, 1, 0)
                elif keyboard.is_pressed('space'):
                    with self.lock:
                        if not self.game_over and not self.paused:
                            self.rotate()
                elif keyboard.is_pressed('down'):
                    with self.lock:
                        if not self.game_over and not self.paused:
                            self.drop()
                elif keyboard.is_pressed('p'):
                    with self.lock:
                        self.paused = not self.paused
                    time.sleep(0.2)
                elif keyboard.is_pressed('r'):
                    with self.lock:
                        self.reset()
                    time.sleep(0.2)
                elif keyboard.is_pressed('q'):
                    self.running = False
                    break
            except:
                pass
            time.sleep(0.05)

    def run(self):
        # Начальный спавн
        self.spawn_shape()
        input_thread = threading.Thread(target=self.handle_input, daemon=True)
        input_thread.start()

        last_update = time.time()
        while self.running:
            now = time.time()
            if now - last_update >= 1.0 / 30:  # 30 FPS
                with self.lock:
                    self.update()
                    self.draw()
                last_update = now
            time.sleep(0.02)

if __name__ == "__main__":
    game = Tetris3D()
    try:
        game.run()
    except KeyboardInterrupt:
        print("\nВыход...")
