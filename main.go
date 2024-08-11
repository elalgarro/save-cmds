package main

import (
    "flag"
    "slices"
    "errors"
	"encoding/json"
	"fmt"
	"io"
	"os"
    "regexp"
	"os/exec"
	"strings"
    "github.com/fatih/color"
    "strconv"
    "github.com/rodaine/table"
)

type SavedCmd struct {
    Alias string `json:"alias"`
    Command string `json:"command"`
}

type  CommandList []SavedCmd

func get_data_directory() string  {

    // check if xdg var is set 
    val := os.Getenv("XDG_DATA_HOME")

    if len(val) == 0{
        // default to the home directory 
        return os.Getenv("HOME") + "/.local/share"
    }
    return val
}

func get_my_cmds_file() (*os.File, error){
    dir := get_data_directory()
    err := os.MkdirAll( dir + "/mycmds" , os.ModePerm)
    if err != nil {
        return nil, err
    }
    full_path := dir + "/mycmds/cmds.json"
    file, err := os.OpenFile(full_path, os.O_RDWR|os.O_CREATE, 0777)
    if err != nil {
        return nil, errors.New("couldn't open or create file")
        
    }
    return file, nil
}

func loadCmdsFromFile(file *os.File) (CommandList ,error){

     commands := make(CommandList, 0)

    byteval, err := io.ReadAll(file)
    
    if err != nil {
        return commands, err
    }
    if err := json.Unmarshal(byteval, &commands ); err != nil{
        return commands, errors.New("Failed to Parse JSON")
    }

    return commands, nil
}

func get_histfile() string {

    histfile := os.Getenv("HISTFILE")
    if len(histfile) == 0 {
        return os.Getenv("HOME") + "/.zsh_history"   
    }

    return histfile
    
}

func extract_cmd( line string) (string, error) {
    _ , command, found := strings.Cut(line,";") 

   if !found {
       return "", errors.New("\n failed to parse line. \n This program expects the history records to be prepended with a timestamp and a semicolon \n however, no semicolon was found.")
   }

 return  command, nil
}



func clear_cmds() error  {
    file, err  := get_my_cmds_file()
    if err != nil {
        return err
    }
    defer file.Close()
    commands := make(CommandList, 0 )
    
    to_write, err := json.Marshal(commands)

    if err != nil {
       return err
    }
    file.Truncate(0)
    file.Seek(0,0)
    file.Write(to_write)
    return nil
}

func add_cmd(alias string) error{
    histfile := get_histfile()
    str, err := exec.Command("bash", "-c", "history -r "+ histfile + "; history 2").Output()
    
    if err != nil {
        return err
    }

    first, _, found  := strings.Cut(string(str), "\n")
    
    if !found {
        return errors.New(" no previous command found, check your historyfile at "+ histfile + "to confirm it is saving correctly" )
    }
    

    file, err  := get_my_cmds_file()
    if err != nil {
        return err
    }
    defer file.Close()  

    command, err := extract_cmd( first )

    commands, err := loadCmdsFromFile(file)

    if  err != nil  {
       return errors.New("config file was not valid JSON.") 
    }

    new_command := SavedCmd{ Command: command, Alias: alias }
   
    commands = append(commands, new_command)

    to_write, err := json.Marshal(commands)

    if err != nil {
        return(err)
    }
    file.Truncate(0)
    file.Seek(0,0)
    file.Write(to_write)
    return nil
}

func listCmds() error {
    file, err  := get_my_cmds_file()
    if err != nil {
        return err
    }
    defer file.Close()

    commands, err := loadCmdsFromFile(file)

    if  err != nil  {
       return err
    }

    fmt.Println("\n")
    if(len(commands) == 0 ) {
        color.Red("No commands saved")
    }else{
        tbl := table.New("Index", "Alias", "Command") 
        for i :=0 ; i < len(commands); i++ {
            tbl.AddRow(strconv.Itoa(i), commands[i].Alias, commands[i].Command)
        }
        tbl.Print()
    }

    fmt.Println("\n")

    return nil
}

func runCmdByIndex(i int) (string, error) {
    file, err  := get_my_cmds_file()
    if err != nil {
        return "", err
    }
    defer file.Close()
    commands, err := loadCmdsFromFile(file)

    if  err != nil  {
       return "", err
    }
    if i >= len(commands) {
        err := errors.New("Attepted to call index of " + strconv.Itoa(i) + " But there are only " + strconv.Itoa(len(commands)) + " saved commands" )
        return "", err
    }
    return  commands[i].Command , nil

}

func tryRunByAlias(arg string) (string, error){
    
    file, err  := get_my_cmds_file()
    if err != nil {
        return "",err
    }
    defer file.Close()
    commands, err := loadCmdsFromFile(file)

    if  err != nil  {
       return "", err
    }
    found := slices.IndexFunc(commands, func( command SavedCmd ) bool {
       return command.Alias == arg
    })
    if found == -1 {
        return  "", errors.New("no matching alias found")
    }
   return commands[found].Command, nil     
}

var alias string  
func init(){
    flag.StringVar(&alias, "a", "", "Alias for the saved command")
}

func firstFlag() int {
    flagIndex  := 1
    for i := 1; i < len(os.Args); i++ {
        flagIndex = i
        if strings.HasPrefix(os.Args[i], "-"){
            break
        }
    }
    return flagIndex
}

func executeCommand(cmd string){
            fmt.Println(cmd)
            splitted := strings.Fields(cmd)

            toRun := exec.Command(splitted[0], splitted[1:]...)
            toRun.Stdin = os.Stdin            
            toRun.Stdout = os.Stdout
            toRun.Stderr = os.Stderr
            err := toRun.Run()
            if err != nil {
                panic(err)
            }
}

func main(){

    flagIndex := firstFlag() 
    flag.CommandLine.Parse(os.Args[flagIndex:])

    if len(os.Args) < 2{
     listCmds()
     return
    }

    arg1 := os.Args[1]

    if arg1 == "add" {
        if err := add_cmd(alias); err != nil {
            os.Stderr.WriteString(err.Error())
        }

    } else if arg1 == "clear" {
        if err := clear_cmds(); err != nil {
            os.Stderr.WriteString(err.Error())
        }
    }else{
        match, _ := regexp.MatchString("[0-9]", arg1)
        if match {
            i , _ := strconv.Atoi(arg1)
            cmd, err := runCmdByIndex(i)
            if err != nil {
                os.Stderr.WriteString("could not find index of "+ arg1)
            }
            executeCommand(cmd)
        }else{
            cmd, err := tryRunByAlias(arg1)
        
            if err != nil {
                os.Stderr.WriteString("No argument named " + arg1 + " and no alias by that name found" )
            }
            executeCommand(cmd)
        }

        
    }
}
