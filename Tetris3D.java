// Tetris3D.java - 3D Тетрис на Java
import org.jline.terminal.Terminal;
import org.jline.terminal.TerminalBuilder;
import org.jline.utils.InfoCmp;
import com.google.gson.*;

import java.io.*;
import java.nio.file.*;
import java.util.*;

public class Tetris3D {
    private static final int WIDTH = 6;
    private static final int DEPTH = 6;
    private static final int HEIGHT = 6;
    private static final int EMPTY = 0;
    private static final String RECORD_FILE = "tetris3d_record.json";

    private static final Map<String, int[][]> SHAPES = new HashMap<>();
    static {
        SHAPES.put("I", new int[][]{{0,0,0},{1,0,0},{2,0,0},{3,0,0}});
        SHAPES.put("O", new int[][]{{0,0,0},{1,0,0},{0,1,0},{1,1,0}});
        SHAPES.put("T", new int[][]{{0,0,0},{1,0,0},{2,0,0},{1,1,0}});
        SHAPES.put("L", new int[][]{{0,0,0},{1,0,0},{2,0,0},{2,1,0}});
        SHAPES.put("J", new int[][]{{0,0,0},{1,0,0},{2,0,0},{0,1,0}});
        SHAPES.put("S", new int[][]{{1,0,0},{2,0,0},{0,1,0},{1,1,0}});
        SHAPES.put("Z", new int[][]{{0,0,0},{1,0,0},{1,1,0},{2,1,0}});
    }

    private Terminal terminal;
    private int[][][] field;
    private int score, speed;
    private boolean gameOver, paused, running;
    private int record;
    private List<int[]> currentShape;
    private int[] shapePos;
    private long dropTimer;
    private boolean lock;

    public Tetris3D() throws IOException {
        terminal = TerminalBuilder.builder().system(true).build();
        resetState();
        record = loadRecord();
        running = true;
        lock = false;
    }

    private void resetState() {
        field = new int[HEIGHT][WIDTH][DEPTH];
        score = 0;
        speed = 1;
        gameOver = false;
        paused = false;
        currentShape = null;
        shapePos = new int[]{0,0,0};
        dropTimer = System.currentTimeMillis();
    }

    private int loadRecord() {
        try {
            String content = new String(Files.readAllBytes(Paths.get(RECORD_FILE)));
            JsonObject obj = new Gson().fromJson(content, JsonObject.class);
            return obj.get("record").getAsInt();
        } catch (Exception e) { return 0; }
    }

    private void saveRecord() {
        try {
            JsonObject obj = new JsonObject();
            obj.addProperty("record", record);
            Files.write(Paths.get(RECORD_FILE), obj.toString().getBytes());
        } catch (Exception e) {}
    }

    private List<int[]> randomShape() {
        String[] names = SHAPES.keySet().toArray(new String[0]);
        String name = names[new Random().nextInt(names.length)];
        int[][] coords = SHAPES.get(name);
        List<int[]> shape = new ArrayList<>();
        for (int[] p : coords) shape.add(p.clone());
        return shape;
    }

    private boolean spawnShape() {
        List<int[]> shape = randomShape();
        int[] xs = shape.stream().mapToInt(p -> p[0]).toArray();
        int[] zs = shape.stream().mapToInt(p -> p[1]).toArray();
        int cx = (Arrays.stream(xs).max().getAsInt() + Arrays.stream(xs).min().getAsInt()) / 2;
        int cz = (Arrays.stream(zs).max().getAsInt() + Arrays.stream(zs).min().getAsInt()) / 2;
        int startX = WIDTH/2 - cx;
        int startZ = DEPTH/2 - cz;
        int startY = HEIGHT - 1;
        for (int[] p : shape) {
            int x = startX + p[0], z = startZ + p[1], y = startY + p[2];
            if (x < 0 || x >= WIDTH || z < 0 || z >= DEPTH || y < 0 || y >= HEIGHT || field[y][x][z] != EMPTY) {
                gameOver = true;
                if (score > record) { record = score; saveRecord(); }
                return false;
            }
        }
        currentShape = shape;
        shapePos = new int[]{startX, startZ, startY};
        return true;
    }

    private boolean collides(List<int[]> shape, int[] pos) {
        for (int[] p : shape) {
            int x = pos[0] + p[0], z = pos[1] + p[1], y = pos[2] + p[2];
            if (x < 0 || x >= WIDTH || z < 0 || z >= DEPTH || y < 0 || y >= HEIGHT) return true;
            if (field[y][x][z] != EMPTY) return true;
        }
        return false;
    }

    private void lockShape() {
        for (int[] p : currentShape) {
            int x = shapePos[0] + p[0], z = shapePos[1] + p[1], y = shapePos[2] + p[2];
            field[y][x][z] = 1;
        }
        int cleared = 0;
        for (int y = 0; y < HEIGHT; y++) {
            boolean full = true;
            for (int x = 0; x < WIDTH; x++) for (int z = 0; z < DEPTH; z++) if (field[y][x][z] == EMPTY) { full = false; break; }
            if (full) {
                for (int y2 = y; y2 > 0; y2--) field[y2] = field[y2-1];
                field[0] = new int[WIDTH][DEPTH];
                cleared++;
            }
        }
        if (cleared > 0) { score += cleared; speed = 1 + score/3; }
        if (!spawnShape()) gameOver = true;
    }

    private boolean move(int dx, int dz, int dy) {
        int[] newPos = {shapePos[0]+dx, shapePos[1]+dz, shapePos[2]+dy};
        if (!collides(currentShape, newPos)) { shapePos = newPos; return true; }
        return false;
    }

    private void rotate() {
        List<int[]> newShape = new ArrayList<>();
        for (int[] p : currentShape) newShape.add(new int[]{-p[1], p[0], p[2]});
        if (!collides(newShape, shapePos)) currentShape = newShape;
    }

    private void drop() {
        while (move(0,0,-1)) {}
        lockShape();
    }

    private void update() {
        if (gameOver || paused) return;
        if (System.currentTimeMillis() - dropTimer >= 1000 / speed) {
            if (!move(0,0,-1)) lockShape();
            dropTimer = System.currentTimeMillis();
        }
    }

    private int[][][] getFieldWithShape() {
        int[][][] copy = new int[HEIGHT][WIDTH][DEPTH];
        for (int y = 0; y < HEIGHT; y++) for (int x = 0; x < WIDTH; x++) copy[y][x] = field[y][x].clone();
        if (currentShape != null) {
            for (int[] p : currentShape) {
                int x = shapePos[0]+p[0], z = shapePos[1]+p[1], y = shapePos[2]+p[2];
                if (x>=0 && x<WIDTH && z>=0 && z<DEPTH && y>=0 && y<HEIGHT) copy[y][x][z] = 2;
            }
        }
        return copy;
    }

    private void draw() {
        terminal.puts(InfoCmp.Capability.clear_screen);
        int[][][] fieldCopy = getFieldWithShape();
        System.out.println("═".repeat(WIDTH*2+10));
        System.out.printf("  Счёт: %d   Рекорд: %d   Скорость: %d%n", score, record, speed);
        System.out.println("═".repeat(WIDTH*2+10));
        for (int y = HEIGHT-1; y >= 0; y--) {
            System.out.printf("  Y=%d  ", y);
            for (int z = 0; z < DEPTH; z++) {
                for (int x = 0; x < WIDTH; x++) {
                    int val = fieldCopy[y][x][z];
                    if (val == 2) System.out.print("\u001B[32m█ \u001B[0m");
                    else if (val == 1) System.out.print("\u001B[36m█ \u001B[0m");
                    else System.out.print("· ");
                }
                System.out.print("  ");
            }
            System.out.println();
        }
        System.out.println("═".repeat(WIDTH*2+10));
        String status = paused ? "ПАУЗА" : gameOver ? "ИГРА ОКОНЧЕНА" : "ИГРА";
        System.out.printf("  %s  |  ←/→ - X  |  W/S - Z  |  Пробел - вращение  |  ↓ - падение%n", status);
        System.out.println("  P - пауза  |  R - рестарт  |  Q - выход");
    }

    private void reset() {
        resetState();
        spawnShape();
    }

    private void handleInput() {
        try {
            while (running) {
                int ch = terminal.reader().read();
                if (ch == -1) continue;
                if (lock) continue;
                lock = true;
                char c = (char) ch;
                switch (c) {
                    case 'a': case 'A': if (!gameOver && !paused) move(-1,0,0); break;
                    case 'd': case 'D': if (!gameOver && !paused) move(1,0,0); break;
                    case 'w': case 'W': if (!gameOver && !paused) move(0,-1,0); break;
                    case 's': case 'S': if (!gameOver && !paused) move(0,1,0); break;
                    case ' ': rotate(); break;
                    case '↓': drop(); break;
                    case 'p': case 'P': paused = !paused; break;
                    case 'r': case 'R': reset(); break;
                    case 'q': case 'Q': running = false; break;
                }
                lock = false;
            }
        } catch (IOException e) {}
    }

    public void run() throws Exception {
        spawnShape();
        Thread inputThread = new Thread(this::handleInput);
        inputThread.setDaemon(true);
        inputThread.start();

        while (running) {
            update();
            draw();
            Thread.sleep(1000/30);
        }
        terminal.close();
    }

    public static void main(String[] args) throws Exception {
        Tetris3D game = new Tetris3D();
        game.run();
    }
}
