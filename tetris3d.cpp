// tetris3d.cpp - 3D Тетрис на C++17
#include <iostream>
#include <vector>
#include <map>
#include <random>
#include <chrono>
#include <thread>
#include <fstream>
#include <string>
#include <algorithm>
#include <cstdlib>

#ifdef _WIN32
    #include <conio.h>
    #include <windows.h>
    #define CLEAR() system("cls")
#else
    #include <ncurses.h>
    #include <termios.h>
    #include <unistd.h>
    #include <fcntl.h>
    #define CLEAR() system("clear")
#endif

const int WIDTH = 6;
const int DEPTH = 6;
const int HEIGHT = 6;
const int EMPTY = 0;
const std::string RECORD_FILE = "tetris3d_record.json";

using Shape = std::vector<std::vector<int>>; // вектор точек [x,y,z]

std::map<std::string, Shape> SHAPES = {
    {"I", {{0,0,0},{1,0,0},{2,0,0},{3,0,0}}},
    {"O", {{0,0,0},{1,0,0},{0,1,0},{1,1,0}}},
    {"T", {{0,0,0},{1,0,0},{2,0,0},{1,1,0}}},
    {"L", {{0,0,0},{1,0,0},{2,0,0},{2,1,0}}},
    {"J", {{0,0,0},{1,0,0},{2,0,0},{0,1,0}}},
    {"S", {{1,0,0},{2,0,0},{0,1,0},{1,1,0}}},
    {"Z", {{0,0,0},{1,0,0},{1,1,0},{2,1,0}}},
};

class Tetris3D {
public:
    Tetris3D() : score(0), speed(1), gameOver(false), paused(false), running(true), lock(false) {
        field = std::vector<std::vector<std::vector<int>>>(HEIGHT, std::vector<std::vector<int>>(WIDTH, std::vector<int>(DEPTH, EMPTY)));
        record = loadRecord();
        dropTimer = std::chrono::steady_clock::now();
        spawnShape();
    }

    ~Tetris3D() {
#ifndef _WIN32
        endwin();
#endif
    }

    void run() {
#ifdef _WIN32
        while (running) {
            handleInputWindows();
            update();
            draw();
            std::this_thread::sleep_for(std::chrono::milliseconds(1000/30));
        }
#else
        initscr();
        raw();
        noecho();
        keypad(stdscr, TRUE);
        nodelay(stdscr, TRUE);
        curs_set(0);
        while (running) {
            handleInputNcurses();
            update();
            drawNcurses();
            napms(1000/30);
        }
        endwin();
#endif
    }

private:
    std::vector<std::vector<std::vector<int>>> field;
    int score, speed;
    bool gameOver, paused, running, lock;
    int record;
    Shape currentShape;
    std::vector<int> shapePos; // [x,z,y]
    std::chrono::steady_clock::time_point dropTimer;
    std::mt19937 rng;

    int loadRecord() {
        std::ifstream in(RECORD_FILE);
        if (!in) return 0;
        std::string content((std::istreambuf_iterator<char>(in)), std::istreambuf_iterator<char>());
        size_t pos = content.find("\"record\":");
        if (pos != std::string::npos) {
            pos += 9;
            size_t end = content.find(",", pos);
            if (end == std::string::npos) end = content.find("}", pos);
            return std::stoi(content.substr(pos, end - pos));
        }
        return 0;
    }

    void saveRecord() {
        std::ofstream out(RECORD_FILE);
        out << "{\"record\":" << record << "}";
    }

    Shape randomShape() {
        std::vector<std::string> keys;
        for (auto& kv : SHAPES) keys.push_back(kv.first);
        std::uniform_int_distribution<int> dist(0, keys.size()-1);
        return SHAPES[keys[dist(rng)]];
    }

    bool spawnShape() {
        Shape shape = randomShape();
        std::vector<int> xs, zs;
        for (auto& p : shape) { xs.push_back(p[0]); zs.push_back(p[1]); }
        int cx = (*std::max_element(xs.begin(), xs.end()) + *std::min_element(xs.begin(), xs.end())) / 2;
        int cz = (*std::max_element(zs.begin(), zs.end()) + *std::min_element(zs.begin(), zs.end())) / 2;
        int startX = WIDTH/2 - cx;
        int startZ = DEPTH/2 - cz;
        int startY = HEIGHT - 1;
        for (auto& p : shape) {
            int x = startX + p[0], z = startZ + p[1], y = startY + p[2];
            if (x < 0 || x >= WIDTH || z < 0 || z >= DEPTH || y < 0 || y >= HEIGHT || field[y][x][z] != EMPTY) {
                gameOver = true;
                if (score > record) { record = score; saveRecord(); }
                return false;
            }
        }
        currentShape = shape;
        shapePos = {startX, startZ, startY};
        return true;
    }

    bool collides(const Shape& shape, const std::vector<int>& pos) {
        for (auto& p : shape) {
            int x = pos[0] + p[0], z = pos[1] + p[1], y = pos[2] + p[2];
            if (x < 0 || x >= WIDTH || z < 0 || z >= DEPTH || y < 0 || y >= HEIGHT) return true;
            if (field[y][x][z] != EMPTY) return true;
        }
        return false;
    }

    void lockShape() {
        for (auto& p : currentShape) {
            int x = shapePos[0] + p[0], z = shapePos[1] + p[1], y = shapePos[2] + p[2];
            field[y][x][z] = 1;
        }
        int cleared = 0;
        for (int y = 0; y < HEIGHT; y++) {
            bool full = true;
            for (int x = 0; x < WIDTH; x++) for (int z = 0; z < DEPTH; z++) if (field[y][x][z] == EMPTY) { full = false; break; }
            if (full) {
                for (int y2 = y; y2 > 0; y2--) field[y2] = field[y2-1];
                field[0] = std::vector<std::vector<int>>(WIDTH, std::vector<int>(DEPTH, EMPTY));
                cleared++;
            }
        }
        if (cleared > 0) { score += cleared; speed = 1 + score/3; }
        if (!spawnShape()) gameOver = true;
    }

    bool move(int dx, int dz, int dy) {
        std::vector<int> newPos = {shapePos[0]+dx, shapePos[1]+dz, shapePos[2]+dy};
        if (!collides(currentShape, newPos)) { shapePos = newPos; return true; }
        return false;
    }

    void rotate() {
        Shape newShape;
        for (auto& p : currentShape) newShape.push_back({-p[1], p[0], p[2]});
        if (!collides(newShape, shapePos)) currentShape = newShape;
    }

    void drop() {
        while (move(0,0,-1)) {}
        lockShape();
    }

    void update() {
        if (gameOver || paused) return;
        auto now = std::chrono::steady_clock::now();
        if (std::chrono::duration_cast<std::chrono::milliseconds>(now - dropTimer).count() >= 1000 / speed) {
            if (!move(0,0,-1)) lockShape();
            dropTimer = now;
        }
    }

    void draw() {
        CLEAR();
        auto fieldCopy = field;
        if (!currentShape.empty()) {
            for (auto& p : currentShape) {
                int x = shapePos[0] + p[0], z = shapePos[1] + p[1], y = shapePos[2] + p[2];
                if (x>=0 && x<WIDTH && z>=0 && z<DEPTH && y>=0 && y<HEIGHT) fieldCopy[y][x][z] = 2;
            }
        }
        std::cout << std::string(WIDTH*2+10, '═') << std::endl;
        printf("  Счёт: %d   Рекорд: %d   Скорость: %d\n", score, record, speed);
        std::cout << std::string(WIDTH*2+10, '═') << std::endl;
        for (int y = HEIGHT-1; y >= 0; y--) {
            printf("  Y=%d  ", y);
            for (int z = 0; z < DEPTH; z++) {
                for (int x = 0; x < WIDTH; x++) {
                    int val = fieldCopy[y][x][z];
                    if (val == 2) std::cout << "\033[32m█ \033[0m";
                    else if (val == 1) std::cout << "\033[36m█ \033[0m";
                    else std::cout << "· ";
                }
                std::cout << "  ";
            }
            std::cout << std::endl;
        }
        std::cout << std::string(WIDTH*2+10, '═') << std::endl;
        std::string status = paused ? "ПАУЗА" : gameOver ? "ИГРА ОКОНЧЕНА" : "ИГРА";
        std::cout << "  " << status << "  |  ←/→ - X  |  W/S - Z  |  Пробел - вращение  |  ↓ - падение" << std::endl;
        std::cout << "  P - пауза  |  R - рестарт  |  Q - выход" << std::endl;
    }

#ifdef _WIN32
    void handleInputWindows() {
        if (_kbhit()) {
            int ch = _getch();
            if (lock) return;
            lock = true;
            if (ch == 224) {
                ch = _getch();
                switch (ch) {
                    case 75: if (!gameOver && !paused) move(-1,0,0); break;
                    case 77: if (!gameOver && !paused) move(1,0,0); break;
                    case 80: if (!gameOver && !paused) drop(); break;
                }
            } else {
                switch (tolower(ch)) {
                    case 'w': if (!gameOver && !paused) move(0,-1,0); break;
                    case 's': if (!gameOver && !paused) move(0,1,0); break;
                    case ' ': if (!gameOver && !paused) rotate(); break;
                    case 'p': paused = !paused; break;
                    case 'r': reset(); break;
                    case 'q': running = false; break;
                }
            }
            lock = false;
        }
    }
#else
    void handleInputNcurses() {
        int ch = getch();
        if (ch == ERR) return;
        if (lock) return;
        lock = true;
        switch (ch) {
            case KEY_LEFT: if (!gameOver && !paused) move(-1,0,0); break;
            case KEY_RIGHT: if (!gameOver && !paused) move(1,0,0); break;
            case KEY_DOWN: if (!gameOver && !paused) drop(); break;
            case 'w': case 'W': if (!gameOver && !paused) move(0,-1,0); break;
            case 's': case 'S': if (!gameOver && !paused) move(0,1,0); break;
            case ' ': if (!gameOver && !paused) rotate(); break;
            case 'p': case 'P': paused = !paused; break;
            case 'r': case 'R': reset(); break;
            case 'q': case 'Q': running = false; break;
        }
        lock = false;
    }
#endif

    void reset() {
        field = std::vector<std::vector<std::vector<int>>>(HEIGHT, std::vector<std::vector<int>>(WIDTH, std::vector<int>(DEPTH, EMPTY)));
        score = 0;
        speed = 1;
        gameOver = false;
        paused = false;
        spawnShape();
    }
};

int main() {
    Tetris3D game;
    game.run();
    return 0;
}
