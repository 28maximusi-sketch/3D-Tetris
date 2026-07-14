// tetris3d.rs - 3D Тетрис на Rust
use crossterm::{
    event::{self, Event, KeyCode},
    execute,
    terminal::{self, Clear, ClearType},
};
use std::io::{stdout, Write};
use std::time::{Duration, Instant};
use serde::{Serialize, Deserialize};
use std::fs;
use rand::Rng;

const WIDTH: usize = 6;
const DEPTH: usize = 6;
const HEIGHT: usize = 6;
const EMPTY: i32 = 0;
const RECORD_FILE: &str = "tetris3d_record.json";

type Shape = Vec<[i32; 3]>; // [x, y, z]

const SHAPES: [(&str, Shape); 7] = [
    ("I", vec![[0,0,0],[1,0,0],[2,0,0],[3,0,0]]),
    ("O", vec![[0,0,0],[1,0,0],[0,1,0],[1,1,0]]),
    ("T", vec![[0,0,0],[1,0,0],[2,0,0],[1,1,0]]),
    ("L", vec![[0,0,0],[1,0,0],[2,0,0],[2,1,0]]),
    ("J", vec![[0,0,0],[1,0,0],[2,0,0],[0,1,0]]),
    ("S", vec![[1,0,0],[2,0,0],[0,1,0],[1,1,0]]),
    ("Z", vec![[0,0,0],[1,0,0],[1,1,0],[2,1,0]]),
];

#[derive(Serialize, Deserialize)]
struct RecordData {
    record: i32,
}

struct Tetris3D {
    field: Vec<Vec<Vec<i32>>>,
    score: i32,
    speed: i32,
    game_over: bool,
    paused: bool,
    running: bool,
    record: i32,
    current_shape: Option<Shape>,
    shape_pos: [i32; 3], // [x, z, y]
    drop_timer: Instant,
    lock_input: bool,
    rng: rand::rngs::ThreadRng,
}

impl Tetris3D {
    fn new() -> Self {
        let mut field = Vec::with_capacity(HEIGHT);
        for _ in 0..HEIGHT {
            let mut plane = Vec::with_capacity(WIDTH);
            for _ in 0..WIDTH {
                plane.push(vec![EMPTY; DEPTH]);
            }
            field.push(plane);
        }
        let record = Self::load_record();
        let mut game = Self {
            field,
            score: 0,
            speed: 1,
            game_over: false,
            paused: false,
            running: true,
            record,
            current_shape: None,
            shape_pos: [0,0,0],
            drop_timer: Instant::now(),
            lock_input: false,
            rng: rand::thread_rng(),
        };
        game.spawn_shape();
        game
    }

    fn load_record() -> i32 {
        if let Ok(data) = fs::read_to_string(RECORD_FILE) {
            if let Ok(rec) = serde_json::from_str::<RecordData>(&data) {
                return rec.record;
            }
        }
        0
    }

    fn save_record(record: i32) {
        let data = RecordData { record };
        let _ = fs::write(RECORD_FILE, serde_json::to_string(&data).unwrap());
    }

    fn random_shape(&mut self) -> Shape {
        let idx = self.rng.gen_range(0..SHAPES.len());
        SHAPES[idx].1.clone()
    }

    fn spawn_shape(&mut self) -> bool {
        let shape = self.random_shape();
        let xs: Vec<i32> = shape.iter().map(|p| p[0]).collect();
        let zs: Vec<i32> = shape.iter().map(|p| p[1]).collect();
        let cx = (*xs.iter().max().unwrap() + *xs.iter().min().unwrap()) / 2;
        let cz = (*zs.iter().max().unwrap() + *zs.iter().min().unwrap()) / 2;
        let start_x = (WIDTH as i32) / 2 - cx;
        let start_z = (DEPTH as i32) / 2 - cz;
        let start_y = (HEIGHT as i32) - 1;
        for p in &shape {
            let x = (start_x + p[0]) as usize;
            let z = (start_z + p[1]) as usize;
            let y = (start_y + p[2]) as usize;
            if x >= WIDTH || z >= DEPTH || y >= HEIGHT || self.field[y][x][z] != EMPTY {
                self.game_over = true;
                if self.score > self.record {
                    self.record = self.score;
                    Self::save_record(self.record);
                }
                return false;
            }
        }
        self.current_shape = Some(shape);
        self.shape_pos = [start_x, start_z, start_y];
        true
    }

    fn collides(&self, shape: &Shape, pos: [i32; 3]) -> bool {
        for p in shape {
            let x = (pos[0] + p[0]) as usize;
            let z = (pos[1] + p[1]) as usize;
            let y = (pos[2] + p[2]) as usize;
            if x >= WIDTH || z >= DEPTH || y >= HEIGHT { return true; }
            if self.field[y][x][z] != EMPTY { return true; }
        }
        false
    }

    fn lock_shape(&mut self) {
        let shape = self.current_shape.take().unwrap();
        for p in &shape {
            let x = (self.shape_pos[0] + p[0]) as usize;
            let z = (self.shape_pos[1] + p[1]) as usize;
            let y = (self.shape_pos[2] + p[2]) as usize;
            self.field[y][x][z] = 1;
        }
        let mut cleared = 0;
        for y in 0..HEIGHT {
            let mut full = true;
            for x in 0..WIDTH {
                for z in 0..DEPTH {
                    if self.field[y][x][z] == EMPTY {
                        full = false;
                        break;
                    }
                }
                if !full { break; }
            }
            if full {
                for y2 in (1..=y).rev() {
                    self.field[y2] = self.field[y2-1].clone();
                }
                self.field[0] = vec![vec![EMPTY; DEPTH]; WIDTH];
                cleared += 1;
            }
        }
        if cleared > 0 {
            self.score += cleared;
            self.speed = 1 + self.score / 3;
        }
        if !self.spawn_shape() {
            self.game_over = true;
        }
    }

    fn move_shape(&mut self, dx: i32, dz: i32, dy: i32) -> bool {
        let new_pos = [self.shape_pos[0] + dx, self.shape_pos[1] + dz, self.shape_pos[2] + dy];
        if let Some(shape) = &self.current_shape {
            if !self.collides(shape, new_pos) {
                self.shape_pos = new_pos;
                return true;
            }
        }
        false
    }

    fn rotate(&mut self) {
        if let Some(shape) = &self.current_shape {
            let new_shape: Shape = shape.iter().map(|p| [-p[1], p[0], p[2]]).collect();
            if !self.collides(&new_shape, self.shape_pos) {
                self.current_shape = Some(new_shape);
            }
        }
    }

    fn drop(&mut self) {
        while self.move_shape(0, 0, -1) {}
        self.lock_shape();
    }

    fn update(&mut self) {
        if self.game_over || self.paused { return; }
        if self.drop_timer.elapsed() >= Duration::from_secs_f64(1.0 / self.speed as f64) {
            if !self.move_shape(0, 0, -1) {
                self.lock_shape();
            }
            self.drop_timer = Instant::now();
        }
    }

    fn get_field_with_shape(&self) -> Vec<Vec<Vec<i32>>> {
        let mut field_copy = self.field.clone();
        if let Some(shape) = &self.current_shape {
            for p in shape {
                let x = (self.shape_pos[0] + p[0]) as usize;
                let z = (self.shape_pos[1] + p[1]) as usize;
                let y = (self.shape_pos[2] + p[2]) as usize;
                if x < WIDTH && z < DEPTH && y < HEIGHT {
                    field_copy[y][x][z] = 2;
                }
            }
        }
        field_copy
    }

    fn draw(&self) {
        execute!(stdout(), Clear(ClearType::All)).unwrap();
        let mut out = stdout();
        let field = self.get_field_with_shape();
        writeln!(out, "{}", "═".repeat(WIDTH*2+10)).unwrap();
        writeln!(out, "  Счёт: {}   Рекорд: {}   Скорость: {}", self.score, self.record, self.speed).unwrap();
        writeln!(out, "{}", "═".repeat(WIDTH*2+10)).unwrap();
        for y in (0..HEIGHT).rev() {
            write!(out, "  Y={}  ", y).unwrap();
            for z in 0..DEPTH {
                for x in 0..WIDTH {
                    let val = field[y][x][z];
                    if val == 2 { write!(out, "\x1b[32m█ \x1b[0m").unwrap(); }
                    else if val == 1 { write!(out, "\x1b[36m█ \x1b[0m").unwrap(); }
                    else { write!(out, "· ").unwrap(); }
                }
                write!(out, "  ").unwrap();
            }
            writeln!(out).unwrap();
        }
        writeln!(out, "{}", "═".repeat(WIDTH*2+10)).unwrap();
        let status = if self.paused { "ПАУЗА" } else if self.game_over { "ИГРА ОКОНЧЕНА" } else { "ИГРА" };
        writeln!(out, "  {}  |  ←/→ - X  |  W/S - Z  |  Пробел - вращение  |  ↓ - падение", status).unwrap();
        writeln!(out, "  P - пауза  |  R - рестарт  |  Q - выход").unwrap();
        out.flush().unwrap();
    }

    fn reset(&mut self) {
        self.field = vec![vec![vec![EMPTY; DEPTH]; WIDTH]; HEIGHT];
        self.score = 0;
        self.speed = 1;
        self.game_over = false;
        self.paused = false;
        self.spawn_shape();
    }

    fn run(&mut self) {
        terminal::enable_raw_mode().unwrap();
        self.drop_timer = Instant::now();

        while self.running {
            // Обработка ввода
            if event::poll(Duration::from_millis(50)).unwrap() {
                if let Event::Key(key) = event::read().unwrap() {
                    if self.lock_input { continue; }
                    self.lock_input = true;
                    match key.code {
                        KeyCode::Left => if !self.game_over && !self.paused { self.move_shape(-1,0,0); }
                        KeyCode::Right => if !self.game_over && !self.paused { self.move_shape(1,0,0); }
                        KeyCode::Down => if !self.game_over && !self.paused { self.drop(); }
                        KeyCode::Up => {} // не используется
                        KeyCode::Char(' ') => if !self.game_over && !self.paused { self.rotate(); }
                        KeyCode::Char('w') | KeyCode::Char('W') => if !self.game_over && !self.paused { self.move_shape(0,-1,0); }
                        KeyCode::Char('s') | KeyCode::Char('S') => if !self.game_over && !self.paused { self.move_shape(0,1,0); }
                        KeyCode::Char('p') | KeyCode::Char('P') => self.paused = !self.paused,
                        KeyCode::Char('r') | KeyCode::Char('R') => self.reset(),
                        KeyCode::Char('q') | KeyCode::Char('Q') => {
                            self.running = false;
                            break;
                        }
                        _ => {}
                    }
                    self.lock_input = false;
                }
            }

            // Обновление и отрисовка
            let now = Instant::now();
            if now - self.last_update >= Duration::from_secs_f64(1.0 / 30.0) {
                self.update();
                self.draw();
                self.last_update = now;
            }
        }
        terminal::disable_raw_mode().unwrap();
    }
}

fn main() {
    let mut game = Tetris3D::new();
    game.run();
    println!("Игра завершена.");
}
