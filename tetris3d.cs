// tetris3d.cs - 3D Тетрис на C#
using System;
using System.Collections.Generic;
using System.IO;
using System.Text.Json;
using System.Threading;

class Tetris3D
{
    private const int WIDTH = 6;
    private const int DEPTH = 6;
    private const int HEIGHT = 6;
    private const int EMPTY = 0;
    private const string RECORD_FILE = "tetris3d_record.json";

    private static readonly Dictionary<string, List<(int x,int y,int z)>> SHAPES = new() {
        {"I", new List<(int,int,int)>{(0,0,0),(1,0,0),(2,0,0),(3,0,0)}},
        {"O", new List<(int,int,int)>{(0,0,0),(1,0,0),(0,1,0),(1,1,0)}},
        {"T", new List<(int,int,int)>{(0,0,0),(1,0,0),(2,0,0),(1,1,0)}},
        {"L", new List<(int,int,int)>{(0,0,0),(1,0,0),(2,0,0),(2,1,0)}},
        {"J", new List<(int,int,int)>{(0,0,0),(1,0,0),(2,0,0),(0,1,0)}},
        {"S", new List<(int,int,int)>{(1,0,0),(2,0,0),(0,1,0),(1,1,0)}},
        {"Z", new List<(int,int,int)>{(0,0,0),(1,0,0),(1,1,0),(2,1,0)}},
    };

    private int[,,] field;
    private int score, speed;
    private bool gameOver, paused, running;
    private int record;
    private List<(int x,int y,int z)> currentShape;
    private (int x, int z, int y) shapePos;
    private DateTime dropTimer;
    private bool lockInput;
    private Random rand;

    public Tetris3D()
    {
        rand = new Random();
        record = LoadRecord();
        ResetState();
        running = true;
    }

    private void ResetState()
    {
        field = new int[HEIGHT, WIDTH, DEPTH];
        score = 0;
        speed = 1;
        gameOver = false;
        paused = false;
        currentShape = null;
        shapePos = (0,0,0);
        dropTimer = DateTime.Now;
        lockInput = false;
        SpawnShape();
    }

    private int LoadRecord()
    {
        if (!File.Exists(RECORD_FILE)) return 0;
        string json = File.ReadAllText(RECORD_FILE);
        var data = JsonSerializer.Deserialize<Dictionary<string,int>>(json);
        return data != null && data.ContainsKey("record") ? data["record"] : 0;
    }

    private void SaveRecord()
    {
        var data = new Dictionary<string,int>{{"record", record}};
        File.WriteAllText(RECORD_FILE, JsonSerializer.Serialize(data));
    }

    private List<(int x,int y,int z)> RandomShape()
    {
        var keys = new List<string>(SHAPES.Keys);
        string name = keys[rand.Next(keys.Count)];
        return new List<(int,int,int)>(SHAPES[name]);
    }

    private bool SpawnShape()
    {
        var shape = RandomShape();
        int[] xs = new int[shape.Count], zs = new int[shape.Count];
        for (int i=0; i<shape.Count; i++) { xs[i]=shape[i].x; zs[i]=shape[i].z; }
        int cx = (Math.Max(xs) + Math.Min(xs)) / 2;
        int cz = (Math.Max(zs) + Math.Min(zs)) / 2;
        int startX = WIDTH/2 - cx;
        int startZ = DEPTH/2 - cz;
        int startY = HEIGHT - 1;
        foreach (var p in shape)
        {
            int x = startX + p.x, z = startZ + p.z, y = startY + p.y;
            if (x<0 || x>=WIDTH || z<0 || z>=DEPTH || y<0 || y>=HEIGHT || field[y,x,z] != EMPTY)
            {
                gameOver = true;
                if (score > record) { record = score; SaveRecord(); }
                return false;
            }
        }
        currentShape = shape;
        shapePos = (startX, startZ, startY);
        return true;
    }

    private bool Collides(List<(int x,int y,int z)> shape, (int x,int z,int y) pos)
    {
        foreach (var p in shape)
        {
            int x = pos.x + p.x, z = pos.z + p.z, y = pos.y + p.y;
            if (x<0 || x>=WIDTH || z<0 || z>=DEPTH || y<0 || y>=HEIGHT) return true;
            if (field[y,x,z] != EMPTY) return true;
        }
        return false;
    }

    private void LockShape()
    {
        foreach (var p in currentShape)
        {
            int x = shapePos.x + p.x, z = shapePos.z + p.z, y = shapePos.y + p.y;
            field[y,x,z] = 1;
        }
        int cleared = 0;
        for (int y = 0; y < HEIGHT; y++)
        {
            bool full = true;
            for (int x=0; x<WIDTH; x++) for (int z=0; z<DEPTH; z++) if (field[y,x,z] == EMPTY) { full=false; break; }
            if (full)
            {
                for (int y2 = y; y2 > 0; y2--) 
                    for (int x=0; x<WIDTH; x++) for (int z=0; z<DEPTH; z++) field[y2,x,z] = field[y2-1,x,z];
                for (int x=0; x<WIDTH; x++) for (int z=0; z<DEPTH; z++) field[0,x,z] = EMPTY;
                cleared++;
            }
        }
        if (cleared > 0) { score += cleared; speed = 1 + score/3; }
        if (!SpawnShape()) gameOver = true;
    }

    private bool Move(int dx, int dz, int dy)
    {
        var newPos = (shapePos.x + dx, shapePos.z + dz, shapePos.y + dy);
        if (!Collides(currentShape, newPos)) { shapePos = newPos; return true; }
        return false;
    }

    private void Rotate()
    {
        var newShape = new List<(int x,int y,int z)>();
        foreach (var p in currentShape) newShape.Add((-p.z, p.x, p.y)); // вращение вокруг Y
        if (!Collides(newShape, shapePos)) currentShape = newShape;
    }

    private void Drop()
    {
        while (Move(0,0,-1)) {}
        LockShape();
    }

    private void Update()
    {
        if (gameOver || paused) return;
        if ((DateTime.Now - dropTimer).TotalMilliseconds >= 1000.0 / speed)
        {
            if (!Move(0,0,-1)) LockShape();
            dropTimer = DateTime.Now;
        }
    }

    private int[,,] GetFieldWithShape()
    {
        var copy = (int[,,])field.Clone();
        if (currentShape != null)
        {
            foreach (var p in currentShape)
            {
                int x = shapePos.x + p.x, z = shapePos.z + p.z, y = shapePos.y + p.y;
                if (x>=0 && x<WIDTH && z>=0 && z<DEPTH && y>=0 && y<HEIGHT) copy[y,x,z] = 2;
            }
        }
        return copy;
    }

    private void Draw()
    {
        Console.Clear();
        var fieldCopy = GetFieldWithShape();
        Console.WriteLine(new string('═', WIDTH*2+10));
        Console.WriteLine($"  Счёт: {score}   Рекорд: {record}   Скорость: {speed}");
        Console.WriteLine(new string('═', WIDTH*2+10));
        for (int y = HEIGHT-1; y >= 0; y--)
        {
            Console.Write($"  Y={y}  ");
            for (int z = 0; z < DEPTH; z++)
            {
                for (int x = 0; x < WIDTH; x++)
                {
                    int val = fieldCopy[y,x,z];
                    if (val == 2) Console.Write("\x1b[32m█ \x1b[0m");
                    else if (val == 1) Console.Write("\x1b[36m█ \x1b[0m");
                    else Console.Write("· ");
                }
                Console.Write("  ");
            }
            Console.WriteLine();
        }
        Console.WriteLine(new string('═', WIDTH*2+10));
        string status = paused ? "ПАУЗА" : gameOver ? "ИГРА ОКОНЧЕНА" : "ИГРА";
        Console.WriteLine($"  {status}  |  ←/→ - X  |  W/S - Z  |  Пробел - вращение  |  ↓ - падение");
        Console.WriteLine("  P - пауза  |  R - рестарт  |  Q - выход");
    }

    private void Reset()
    {
        ResetState();
    }

    public void Run()
    {
        while (running)
        {
            // Ввод
            while (Console.KeyAvailable)
            {
                if (lockInput) continue;
                lockInput = true;
                var key = Console.ReadKey(true);
                switch (key.Key)
                {
                    case ConsoleKey.LeftArrow: if (!gameOver && !paused) Move(-1,0,0); break;
                    case ConsoleKey.RightArrow: if (!gameOver && !paused) Move(1,0,0); break;
                    case ConsoleKey.W: if (!gameOver && !paused) Move(0,-1,0); break;
                    case ConsoleKey.S: if (!gameOver && !paused) Move(0,1,0); break;
                    case ConsoleKey.Spacebar: if (!gameOver && !paused) Rotate(); break;
                    case ConsoleKey.DownArrow: if (!gameOver && !paused) Drop(); break;
                    case ConsoleKey.P: paused = !paused; break;
                    case ConsoleKey.R: Reset(); break;
                    case ConsoleKey.Q: running = false; break;
                }
                lockInput = false;
            }

            Update();
            Draw();
            Thread.Sleep(1000/30);
        }
    }

    static void Main()
    {
        var game = new Tetris3D();
        game.Run();
        Console.WriteLine("Игра завершена.");
    }
}
