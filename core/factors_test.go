package core
import ("math";"testing")
func TestSelection(t *testing.T){
 xs:=[]Observation{{"A",0.4,0.2,0.3},{"B",0.1,0.4,0.2},{"C",-0.2,0.1,-0.1}}
 out,err:=Select(xs,FactorWeights{50,30,20},2,true)
 if err!=nil||len(out)!=2{t.Fatalf("expected two ranked assets: %v %v",out,err)}
 if out[0].Symbol!="A"||math.Abs(out[0].Weight+out[1].Weight-1)>1e-9{t.Fatal("unexpected ranking or normalized weights")}
}
func TestRejectNonfinite(t *testing.T){
 _,e:=Select([]Observation{{"A",math.NaN(),0,0}},FactorWeights{1,0,0},1,false)
 if e==nil{t.Fatal("expected an error for a nonfinite input")}
}
func TestPaperTrade(t *testing.T){
 cash,shares,e:=PaperTrade(1000,0,100,2,5,true)
 if e!=nil||math.Abs(cash-799.9)>1e-9||shares!=2{t.Fatalf("buy failed: %g %g %v",cash,shares,e)}
 cash,shares,e=PaperTrade(cash,shares,105,1,5,false)
 if e!=nil||math.Abs(cash-904.8475)>1e-9||shares!=1{t.Fatalf("sell failed: %g %g %v",cash,shares,e)}
}
